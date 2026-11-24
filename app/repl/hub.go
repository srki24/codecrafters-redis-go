package repl

import (
	"slices"
	"sync"
)

type Hub struct {
	mu       sync.Mutex
	channels []chan Serializeable
	acks     map[int]int
}

func NewHub() Hub {
	acks := make(map[int]int)
	channels := make([]chan Serializeable, 0)
	return Hub{channels: channels, acks: acks}
}

type Serializeable interface {
	Serialize() []byte
}

func (h *Hub) AddChannel(ch chan Serializeable) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.channels = append(h.channels, ch)
}

func (h *Hub) RemoveChannel(ch chan Serializeable) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.channels = slices.DeleteFunc(h.channels, func(c chan Serializeable) bool {
		return c == ch
	})
}

func (h *Hub) Publish(v Serializeable) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ch := range h.channels {
		ch <- v
	}
}

func (h *Hub) NumberOfReplica() int {
	return len(h.channels)
}

func (h *Hub) SetAck(port, offset int) {
	h.mu.Lock()
	h.acks[port] = offset
	h.mu.Unlock()
}

func (h *Hub) GetAck() map[int]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	// copy acks to keep them under a lock while counted.
	out := make(map[int]int, len(h.acks))
	for k, v := range h.acks {
		out[k] = v
	}
	return out
}
