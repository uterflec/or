package agent

import (
	"sync"

	"github.com/ktsoator/or/internal/llm"
)

// messageQueue holds messages enqueued from outside a run (steering or
// follow-up) until the loop drains them at a safe point. It is safe for
// concurrent use: Steer/FollowUp may be called while a run is in flight on
// another goroutine.
type messageQueue struct {
	mu       sync.Mutex
	messages []llm.Message
}

func (q *messageQueue) enqueue(message llm.Message) {
	q.mu.Lock()
	q.messages = append(q.messages, message)
	q.mu.Unlock()
}

// drain removes and returns all queued messages.
func (q *messageQueue) drain() []llm.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.messages) == 0 {
		return nil
	}
	drained := q.messages
	q.messages = nil
	return drained
}

func (q *messageQueue) clear() {
	q.mu.Lock()
	q.messages = nil
	q.mu.Unlock()
}
