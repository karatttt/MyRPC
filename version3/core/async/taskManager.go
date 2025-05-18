package async

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
	"math"
)

// Task 表示一个异步任务
type Task struct {
	MethodName  string             // 方法名
	Request      interface{}        // 请求参数
	RetryTimes   int                // 当前已重试次数
	MaxRetries   int                // 最大重试次数
	Timeout      time.Duration      // 单次任务超时时间
	NextRetryAt  time.Time         // 下次重试时间（用于堆排序）
	ExecuteFunc  func() error       // 任务执行函数
	Status       TaskStatus        // 状态字段
	OnComplete   func(error)       // 最终完成回调
	mu           sync.Mutex // 保证状态变更的线程安全
}
type TaskStatus int

const (
    TaskStatusPending TaskStatus = iota // 等待执行或重试
    TaskStatusRunning                   // 正在执行
    TaskStatusCompleted                 // 已完成（成功或最终失败）
    TaskStatusCanceled                  // 已取消（按需可选）
)

// TaskManager 全局任务管理器
type TaskManager struct {
	tasks     *PriorityQueue  // 基于下次重试时间的优先队列
	mu        sync.Mutex
	wakeChan  chan struct{}   // 用于唤醒定时协程
	closeChan chan struct{}   // 关闭信号
}

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		tasks:     &PriorityQueue{},
		wakeChan:  make(chan struct{}, 1),
		closeChan: make(chan struct{}),
	}
	heap.Init(tm.tasks)
	go tm.scanLoop() // 启动回扫协程
	return tm
}

// 获取全局唯一的任务管理器实例，考虑到多线程安全
var globalTaskManager *TaskManager
var once sync.Once
func GetGlobalTaskManager() *TaskManager {
	once.Do(func() {
		globalTaskManager = NewTaskManager()
	})
	return globalTaskManager
}

// AddTask 添加异步任务
func (tm *TaskManager) AddTask(task *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task.NextRetryAt = time.Now().Add(task.Timeout)
	heap.Push(tm.tasks, task)
	tm.notifyScanner() // 通知回扫协程
}

// 通知回扫协程检查任务
func (tm *TaskManager) notifyScanner() {
	select {
	case tm.wakeChan <- struct{}{}:
	default: // 避免阻塞
	}
}

// 扫描循环（核心逻辑）
func (tm *TaskManager) scanLoop() {
	for {
		select {
		case <-tm.closeChan:
			return
		default:
			tm.processTasks()
		}
	}
}

// 处理超时任务
func (tm *TaskManager) processTasks() {
	tm.mu.Lock()
	if tm.tasks.Len() == 0 {
		tm.mu.Unlock()
		// 无任务时休眠，直到被唤醒
		select {
		case <-tm.wakeChan:
		case <-time.After(10 * time.Second): // 防止长期阻塞
		}
		return
	}

	// 检查堆顶任务是否超时
	now := time.Now()
	task := (*tm.tasks)[0]
	if now.Before(task.NextRetryAt) {
		// 未超时，休眠到最近任务到期
		tm.mu.Unlock()
		time.Sleep(task.NextRetryAt.Sub(now))
		return
	}

	// 弹出超时任务
	task = heap.Pop(tm.tasks).(*Task)
	tm.mu.Unlock()

	// 执行重试逻辑
	go tm.retryTask(task)
}

// 重试任务
func (tm *TaskManager) retryTask(task *Task) {
	task.mu.Lock()
    // 检查状态：如果任务已结束，直接返回，不用再次入队列
    if task.Status != TaskStatusPending {
        task.mu.Unlock()
        return
    }
    task.Status = TaskStatusRunning // 标记为执行中
    task.mu.Unlock()

	err := task.ExecuteFunc()
	if err == nil {
		task.OnComplete(nil)
		return
	}

	// 检查是否达到最大重试次数
	task.RetryTimes++
	if task.RetryTimes > task.MaxRetries {
		// 打印
		fmt.Println("request retry times exceed max retry times")
		task.OnComplete(err)
		return
	}

	// 计算下次重试时间（如指数退避）
	delay := time.Duration(math.Pow(2, float64(task.RetryTimes))) * time.Second
	task.NextRetryAt = time.Now().Add(delay) 
	
	// 重新加入队列
	// 打印重试次数
	fmt.Println("request retry time : ", task.RetryTimes)
	tm.mu.Lock()
	heap.Push(tm.tasks, task)
	task.Status = TaskStatusPending // 恢复状态
	tm.mu.Unlock()
	tm.notifyScanner()
}

// 关闭任务管理器
func (tm *TaskManager) Close() {
	close(tm.closeChan)
}