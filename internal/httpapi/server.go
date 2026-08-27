package httpapi

import (
	"context"
	"net"
	"net/http"
	"time"
)

func NewHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
	}
}

func ServeUntil(ctx context.Context, listener net.Listener, server *http.Server) error {
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return err
		}
		err := <-done
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
