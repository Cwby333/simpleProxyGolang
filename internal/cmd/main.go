package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cwby333/simpleProxyGolang/internal/proxy"
	"golang.org/x/sync/errgroup"
)

const (
	defaultTimeoutShutdown = time.Second * 10
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	addresses := []string{"localhost:8081", "localhost:8082", "localhost:8083"}

	proxy := proxy.New(addresses, "localhost:9000")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		slog.Info("signal received", slog.String("signal", sig.String()))
		cancel()
	}()

	group, gctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		slog.Info("start server")
		return proxy.StartServer(ctx)
	})

	group.Go(func() error {
		<-gctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutShutdown)
		defer cancel()

		slog.Info("start shutdown")

		return proxy.Shutdown(ctx)
	})

	if err := group.Wait(); err != nil {
		slog.Info(err.Error())
	}
}
