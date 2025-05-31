//go:build linux
// +build linux
package poller

import (
	"runtime"
	"sync/atomic"
)

var (
	pollmanager = newManager(runtime.GOMAXPROCS(0)/20 + 1) // pollmanager manage all pollers
)


// manager 管理多个 poll 实例
type manager struct {
	polls    []Poll
	numLoops int32
	pickIdx  int32
}

func newManager(numLoops int) *manager {
	manager := &manager{
		numLoops: int32(numLoops),
	}
	manager.InitManager(numLoops)
	return manager
}

// Init 初始化并创建 poll 数组
func (m *manager) InitManager(numPolls int) error {

	if numPolls < 1 {
		numPolls = 1
	}
	atomic.StoreInt32(&m.numLoops, int32(numPolls))
	m.polls = make([]Poll, numPolls)
	for i := 0; i < numPolls; i++ {
		poll, err := openPoll()
		if err != nil {
			return err
		}
		m.polls[i] = poll
		go poll.Wait()
	}
	return nil
}


// Pick 轮询选择一个 poller（简单轮询实现）
func (m *manager) Pick() Poll {
	num := int(atomic.LoadInt32(&m.numLoops))
	if num == 0 {
		return nil
	}
	// 使用原子递增实现简单轮询
	idx := int(atomic.AddInt32(&m.pickIdx, 1)) % num
	return m.polls[idx]
}

// openPoll 是一个工厂函数，需你自己实现
func openPoll() (Poll, error) {
	// TODO: 实现你的 Poll 创建逻辑
	return nil, nil
}
