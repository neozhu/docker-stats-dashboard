package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/your-org/docker-stats-dashboard/agent/internal/stream"
)

type Server struct {
	hub    *stream.Hub
	logger *slog.Logger
	srv    *http.Server
}

func NewServer(logger *slog.Logger, listenAddr string, hub *stream.Hub) *Server {
	mux := http.NewServeMux()
	s := &Server{
		hub:    hub,
		logger: logger,
	}

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.hub.ServeWS(w, r)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	s.srv = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutdownCtx)
	}()

	s.logger.Info("websocket server listening", slog.String("addr", l.Addr().String()))
	if err := s.srv.Serve(l); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
