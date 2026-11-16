package repl

import (
	"slices"
	"sync"
)

type Hub struct {
	mu       sync.Mutex
	channels []chan Serializeable
}

type Serializeable interface {
	Serialize() []byte
}

func (rh *Hub) AddChannel(ch chan Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = append(rh.channels, ch)
}

func (rh *Hub) RemoveChannel(ch chan Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = slices.DeleteFunc(rh.channels, func(c chan Serializeable) bool {
		return c == ch
	})
}

func (rh *Hub) Publish(v Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	for _, ch := range rh.channels {
		ch <- v
	}
}

func (rh *Hub) NumberOfReplica() int {
	return len(rh.channels)
}
