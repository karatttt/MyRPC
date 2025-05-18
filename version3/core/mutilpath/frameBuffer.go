package mutilpath
import (
	"container/list"
	"fmt"
	"sync"
)


type frameBuffer struct {
	mu         sync.Mutex
	frames     map[uint64]*list.List // RequestID -> 帧链表
	currentLen int
	maxLen     int
}


func (fb *frameBuffer) store(frame MuxFrame, requestID uint64) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if fb.currentLen >= fb.maxLen {
		return fmt.Errorf("frame buffer is full")
	}

	if _, exists := fb.frames[requestID]; !exists {
		fb.frames[requestID] = list.New()
	}
	fb.frames[requestID].PushBack(frame)
	fb.currentLen++
	return nil
}

func (fb *frameBuffer) retrieve(reqID uint64) ([]MuxFrame, bool) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	frames, exists := fb.frames[reqID]
	if !exists {
		return nil, false
	}

	result := make([]MuxFrame, 0, frames.Len())
	for e := frames.Front(); e != nil; e = e.Next() {
		result = append(result, e.Value.(MuxFrame))
	}

	delete(fb.frames, reqID)
	fb.currentLen -= len(result)
	return result, true
}


