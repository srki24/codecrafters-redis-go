package sub

import (
	"fmt"
	"slices"
	"sync"
)

type Hub struct {
	mu   sync.Mutex
	subs map[string][]chan []byte
}

func NewHub() Hub {
	subs := make(map[string][]chan []byte, 0)
	return Hub{subs: subs}
}

func (h *Hub) AddChannel(sub string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if s, ok := h.subs[sub]; ok {
		s = append(s, ch)
		h.subs[sub] = s
	} else {
		h.subs[sub] = append([]chan []byte{}, ch)
	}
	fmt.Println(len(h.subs[sub]))
}

func (h *Hub) RemoveChannel(sub string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subs[sub] = slices.DeleteFunc(
		h.subs[sub],
		func(c chan []byte) bool {
			return c == ch
		})
	if len(h.subs[sub]) == 0 {
		delete(h.subs, sub)
	}
}

func (h *Hub) NumberOfSubscribers(sub string) int {
	chs := h.subs[sub]
	return len(chs)

}

func (h *Hub) Publish(msg []byte, sub string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	chs, ok := h.subs[sub]
	if !ok {
		return fmt.Errorf("non existing channel: %s", sub)
	}
	for _, ch := range chs {
		ch <- msg
	}
	return nil
}
