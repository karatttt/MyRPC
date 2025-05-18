	package internel

	import (
		"MyRPC/common"
		"context"
		"sync"
		"time"
		"MyRPC/core/async"
	)

	// Future 表示一个异步操作的结果
	type Future struct {
		mu        sync.Mutex
		done      chan struct{}
		result    interface{}
		err       *common.RPCError
		callbacks []func(interface{}, *common.RPCError)
		Task      *async.Task // 关联的异步任务
	}

	// NewFuture 创建一个新的Future实例
	func NewFuture() *Future {
		return &Future{
			done:      make(chan struct{}),
			callbacks: make([]func(interface{}, *common.RPCError), 0),
		}
	}

	// SetResult 设置Future的结果
	func (f *Future) SetResult(result interface{}, err *common.RPCError) {
		f.mu.Lock()
		defer f.mu.Unlock()

		if f.isDone() {
			return
		}

		f.result = result
		f.Task.Status = async.TaskStatusCompleted
		f.err = err
		close(f.done)

		// 执行所有注册的回调
		for _, callback := range f.callbacks {
			callback(result, err)
		}
	}

	// Get 获取Future的结果，会阻塞直到结果可用
	func (f *Future) Get() (interface{}, *common.RPCError) {
		<-f.done
		return f.result, f.err
	}

	// GetWithTimeout 带超时的获取Future的结果
	func (f *Future) GetWithTimeout(timeout time.Duration) (interface{}, *common.RPCError) {
		select {
		case <-f.done:
			return f.result, f.err
		case <-time.After(timeout):
			return nil, &common.RPCError{
				Code:    common.ErrCodeTimeout,
				Message: "future get timeout",
			}
		}
	}

	// GetWithContext 使用context获取Future的结果
	func (f *Future) GetWithContext(ctx context.Context) (interface{}, *common.RPCError) {
		select {
		case <-f.done:
			return f.result, f.err
		case <-ctx.Done():
			return nil, &common.RPCError{
				Code:    common.ErrCodeTimeout,
				Message: "future get cancelled by context",
			}
		}
	}

	// OnComplete 注册一个回调函数，当Future完成时会被调用
	func (f *Future) OnComplete(callback func(interface{}, *common.RPCError)) {
		f.mu.Lock()
		defer f.mu.Unlock()

		if f.isDone() {
			// 如果已经完成，立即执行回调
			callback(f.result, f.err)
			return
		}

		f.callbacks = append(f.callbacks, callback)
	}

	// IsDone 检查Future是否已完成
	func (f *Future) isDone() bool {
		select {
		case <-f.done:
			return true
		default:
			return false
		}	
	}

	// IsDone 公开的检查方法
	func (f *Future) IsDone() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.isDone()
	}
