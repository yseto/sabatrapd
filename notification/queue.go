package notification

import (
	"container/list"
	"sync"
)

type Queue struct {
	q *list.List
	m sync.Mutex
}

type Item struct {
	OccurredAt int64
	Addr       string
	Message    string
}

func NewQueue() *Queue {
	return &Queue{q: list.New()}
}

func (q *Queue) Enqueue(item Item) {
	q.m.Lock()
	q.q.PushBack(item)
	q.m.Unlock()
}

func (q *Queue) Dequeue() {
	e := q.q.Front()
	// val := e.Value.(Item)
	q.m.Lock()
	q.q.Remove(e)
	q.m.Unlock()
}
