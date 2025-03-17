package limiter

import (
	"errors"
	"sync"
	"time"
)

//leaky bucket
type Limiter struct {
	mu sync.Mutex
	clients map[string]*Client
	ttl time.Duration
	maxSize int

	maxGeneralSize int
	generalSize int
}

type Client struct {
	IP string
	mu sync.Mutex
	ttl time.Duration
	currentSize int
}

func New(maxSize, maxGeneralSize int, ttl time.Duration) *Limiter {
	return &Limiter{
		mu: sync.Mutex{},
		clients: make(map[string]*Client),
		ttl: ttl,
		maxSize: maxSize,
		maxGeneralSize: maxGeneralSize,
	}
} 

func newClient(ttl time.Duration, IP string) *Client {
	return &Client{
		IP: IP,
		ttl: ttl,
		currentSize: 0,
		mu: sync.Mutex{},
	}
}

func (l *Limiter) Allow(IP string) error {	
	l.mu.Lock()
	defer l.mu.Unlock()

	client, ok := l.clients[IP]	

	if !ok {
		client = newClient(l.ttl, IP)

		l.clients[IP] = client
	}

	if client.currentSize >= l.maxSize {
		return errors.New("forbidden")
	}

	if l.generalSize >= l.maxGeneralSize {
		return errors.New("forbidden")
	}

	client.currentSize++
	l.generalSize++	

	return nil
}