package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/benzhi/oral-history-release/internal/application"
	"github.com/benzhi/oral-history-release/internal/httpapi"
	"github.com/benzhi/oral-history-release/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Printf("服务退出：%v", err)
		os.Exit(1)
	}
}

func run() error {
	configuration, err := loadConfig()
	if err != nil {
		return err
	}
	dataDir := configuration.DataDir
	cleanup := func() {}
	if configuration.Selfcheck {
		dataDir, err = os.MkdirTemp("", "oral-history-selfcheck-*")
		if err != nil {
			return err
		}
		cleanup = func() { _ = os.RemoveAll(dataDir) }
		defer cleanup()
	}
	repository, err := store.Open(dataDir)
	if err != nil {
		return fmt.Errorf("恢复存储: %w", err)
	}
	service := application.NewService(repository)
	api := httpapi.New(service)
	listener, err := net.Listen("tcp", configuration.Address)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", configuration.Address, err)
	}
	server := httpapi.NewHTTPServer(api.Handler())
	if configuration.Selfcheck {
		return runSelfcheck(listener, server)
	}
	log.Printf("口述史资料公开授权工作台已监听 http://%s", listener.Addr())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := httpapi.ServeUntil(ctx, listener, server); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
