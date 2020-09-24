package queue

import (
	"sync/atomic"
	"unsafe"
)

// LFQueue is a lock-free queue
type LFQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

// node 队列节点
type node struct {
	value interface{}
	next  unsafe.Pointer
}

// NewLFQueue
func NewLFQueue() *LFQueue {
	n := unsafe.Pointer(&node{})
	return &LFQueue{head: n, tail: n}
}

// Enqueue 入队
func (q *LFQueue) Enqueue(v interface{}) {
	n := &node{value: v}
	for {
		tail := load(&q.tail)
		next := load(&tail.next)
		// tail is tail
		if tail == load(&q.tail) {
			if next == nil {
				// 将next节点设为 n
				// [------(tail)]---nil(next)
				// [------(tail)]---n(next)
				if cas(&tail.next, next, n) {
					// 同时将tail移到尾部
					// [------(tail)]---n(next)
					// [------n]---(tail)
					cas(&q.tail, tail, n)
					return
				}
				// tail was not pointing to the last node
				// 说明在其他线程中有入队操作
			} else {
				// 将 tail 移到尾部就行了
				//               next
				// [---tail]---<another>
				// [---<another>]---tail
				cas(&q.tail, tail, next)
			}
		}
	}
}

// Dequeue 出队
func (q *LFQueue) Dequeue() interface{} {
	for {
		head := load(&q.head)
		tail := load(&q.tail)
		next := load(&head.next)
		// head is head
		if head == load(&q.head) {
			// queue empty or tail falling behind
			if head == tail {
				if next == nil {
					return nil
				}
				// 或者因为有其他线程的操作，导致tail不在尾部
				// 将tail后移
				cas(&q.tail, tail, next)
			} else {
				// 获取next的值
				v := next.value
				// 交换 head & next（head后移）
				if cas(&q.head, head, next) {
					return v
				}
			}
		}
	}
}

func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

// cas
func cas(p *unsafe.Pointer, old, new *node) (ok bool) {
	// cas 原子操作
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}
