package queue

import "sync"

// LKQueue 使用锁来实现队列
type LKQueue struct {
	data []interface{}
	mu   sync.Mutex
}

// NewLKQueue
func NewLKQueue(n int) (q *LKQueue) {
	return &LKQueue{data: make([]interface{}, 0, n)}
}

// Enqueue 入队
func (q *LKQueue) Enqueue(v interface{}) {
	q.mu.Lock()
	q.data = append(q.data, v)
	q.mu.Unlock()
}

// Dequeue 出队
func (q *LKQueue) Dequeue() interface{} {
	q.mu.Lock()
	// 如果队列为空，nil
	if len(q.data) == 0 {
		q.mu.Unlock()
		return nil
	}
	v := q.data[0]
	q.data = q.data[1:]
	q.mu.Unlock()
	return v
}
