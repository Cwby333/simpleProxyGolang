package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Proxy struct {
	mu sync.RWMutex
	counter int
	addresses []string
	
	server http.Server
}

func New(addresses []string, address string) *Proxy {
	return &Proxy{
		mu: sync.RWMutex{},
		addresses: addresses,
		server: http.Server{
			Addr: address,
		},
	}
}

func (p *Proxy) proxyHandle(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.NewString() 	
	t := time.Now()
	defer func ()  {
		slog.Info("duration request", 
		slog.String("duration", time.Since(t).String()), slog.String("requestID", requestID))
	}()

	slog.Info(
		"proxy", 
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("protocol", r.Proto),
		slog.String("IP", r.RemoteAddr),
		slog.Int64("content-length", r.ContentLength),
		slog.String("requestID", requestID),
	)

	p.mu.Lock()
	var counter int
	switch p.counter {
	case len(p.addresses):
		counter = 0
		p.counter = 0
	default:
		counter = p.counter
		p.counter++
	}
	p.mu.Unlock()

	p.mu.RLock()
	address := p.addresses[counter]
	p.mu.RUnlock()

	m := r.Header.Clone()
	
	req ,err := http.NewRequest(
		r.Method, 
		fmt.Sprintf("http://%s%s", address, r.URL), r.Body,
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	req.Header = m
	req.Header.Add("X-REQUEST-ID", requestID)

	client := http.Client{}
	
	response, err := client.Do(req)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	response.Body.Close()

	header := response.Header
	for key, value := range header {
		for i := range value {
			w.Header().Add(key, value[i])
		}
	}

	w.WriteHeader(response.StatusCode)
	w.Write(data)
}

func (p *Proxy) StartServer() error {
	mux := http.NewServeMux()

	mux.Handle("/", http.HandlerFunc(p.proxyHandle))

	p.server.Handler = mux

	return p.server.ListenAndServe()
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}