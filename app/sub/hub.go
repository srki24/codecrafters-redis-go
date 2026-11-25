package sub

import "sync"

type Hub struct {
	mu       sync.Mutex
	channels []chan string
}
