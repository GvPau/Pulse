package scheduler

import (
	"container/heap"
	"time"
	"uuid"
)

type entry struct {
	monitorID uuid.UUID
	nextRun   time.Time
	index     int
}

type queue struct {
	entries []*entry
	byID    map[uuid.UUID]*entry
}

func (q *queue) Len() int {
	return len(q.entries)
}

func (q *queue) Less(i, j int) bool {
	return q.entries[i].nextRun.Before(q.entries[j].nextRun)
}

func (q *queue) Swap(i, j int) {
	q.entries[i], q.entries[j] = q.entries[j], q.entries[i]
	q.entries[i].index = i
	q.entries[j].index = j
}

func (q *queue) Push(x interface{}) {
	e := x.(*entry)
	e.index = len(q.entries)
	q.entries = append(q.entries, e)
	q.byID[e.monitorID] = e
}

func (q *queue) Pop() any {
	old := q.entries
	n := len(old)
	e := old[n-1]
	q.entries = old[0 : n-1]
	delete(q.byID, e.monitorID)
	return e
}

func newQueue() *queue {
	return &queue{
		entries: []*entry{},
		byID:    make(map[uuid.UUID]*entry),
	}
}

func (q *queue) top() *entry {
	if len(q.entries) == 0 {
		return nil
	}
	return q.entries[0]
}

func (q *queue) push(e *entry) {
	heap.Push(q, e)
}

func (q *queue) update(monitorID uuid.UUID, nextRun time.Time) {
	e, ok := q.byID[monitorID]
	if !ok {
		return
	}

	e.nextRun = nextRun
	heap.Fix(q, e.index)
}

func (q *queue) remove(monitorID uuid.UUID) {
	e, ok := q.byID[monitorID]
	if !ok {
		return
	}
	heap.Remove(q, e.index)
}
