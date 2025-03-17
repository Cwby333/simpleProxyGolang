package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

//leaky bucket
type Limiter struct {
	mu      sync.Mutex
	clients map[string]*Client
	ttl     time.Duration
	maxSize int

	mainCtx context.Context

	maxGeneralSize int
	generalSize    int
}

type Client struct {
	IP  string
	mu  sync.Mutex
	ttl time.Duration
	ch  chan struct{}
}

func New(ctx context.Context, maxSize, maxGeneralSize int, ttl time.Duration) *Limiter {
	return &Limiter{
		mu:             sync.Mutex{},
		mainCtx:        ctx,
		clients:        make(map[string]*Client),
		ttl:            ttl,
		maxSize:        maxSize,
		maxGeneralSize: maxGeneralSize,
	}
}

func newClient(ttl time.Duration, IP string, maxSize int) *Client {
	return &Client{
		IP:  IP,
		ttl: ttl,
		ch:  make(chan struct{}, maxSize),
		mu:  sync.Mutex{},
	}
}

func (c *Client) startRenewal(ctx context.Context) {
	ticker := time.NewTicker(c.ttl)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			<-c.ch
		}
	}
}

func (l *Limiter) Allow(IP string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	client, ok := l.clients[IP]

	if !ok {
		client = newClient(l.ttl, IP, l.maxSize)

		l.clients[IP] = client
		go func() {
			client.startRenewal(l.mainCtx)
		}()
	}

	select {
	case client.ch <- struct{}{}:
	default:
		return errors.New("forbidden")
	}

	if l.generalSize >= l.maxGeneralSize {
		return errors.New("forbidden")
	}

	l.generalSize++

	return nil
}
