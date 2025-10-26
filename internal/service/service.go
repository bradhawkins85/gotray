package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/example/gotray/internal/config"
	"github.com/example/gotray/internal/ipc"
	"github.com/example/gotray/internal/menu"
	"github.com/example/gotray/internal/protocol"
	"github.com/example/gotray/internal/security"
)

// Service coordinates the system-level background process that brokers menu
// state to user-session tray agents.
type Service struct {
	token    string
	endpoint ipc.Endpoint

	mu  sync.RWMutex
	cfg *config.Config
}

// New constructs a Service that provides the current tray configuration.
func New() (*Service, error) {
	srv := &Service{
		token:    security.ResolveServiceToken(),
		endpoint: ipc.DefaultEndpoint(),
	}
	if srv.token == "" {
		return nil, fmt.Errorf("service token could not be resolved; set GOTRAY_SERVICE_TOKEN")
	}

	if _, err := srv.currentConfig(); err != nil {
		return nil, err
	}
	return srv, nil
}

// Endpoint exposes the listening endpoint for logging and diagnostics.
func (s *Service) Endpoint() string {
	return s.endpoint.String()
}

// Run starts the listener and serves requests until the context is canceled.
func (s *Service) Run(ctx context.Context) error {
	listener, err := s.endpoint.Listen()
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.endpoint.String(), err)
	}
	defer listener.Close()

	log.Printf("GoTray service listening on %s", s.endpoint.String())

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("GoTray service shutting down")
				return context.Canceled
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("temporary accept error: %v", err)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return fmt.Errorf("accept connection: %w", err)
		}

		go s.handleConnection(ctx, conn)
	}
}

func (s *Service) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req protocol.Request
	if err := decoder.Decode(&req); err != nil {
		log.Printf("service: failed to decode request: %v", err)
		return
	}

	if !s.authorize(req.Token) {
		_ = encoder.Encode(protocol.Response{Error: "unauthorized"})
		return
	}

	switch req.Command {
	case protocol.CommandMenuGet:
		cfg, err := s.currentConfig()
		if err != nil {
			_ = encoder.Encode(protocol.Response{Error: err.Error()})
			return
		}
		_ = encoder.Encode(protocol.Response{Items: cfg.Items})
	default:
		_ = encoder.Encode(protocol.Response{Error: fmt.Sprintf("unknown command: %s", req.Command)})
	}
}

func (s *Service) authorize(token string) bool {
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) == 1
}

func (s *Service) currentConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Items) == 0 {
		cfg.Items = menu.DefaultItems()
		menu.EnsureSequentialOrder(&cfg.Items)
		if err := config.Save(cfg); err != nil {
			return nil, fmt.Errorf("seed defaults: %w", err)
		}
	} else {
		menu.EnsureSequentialOrder(&cfg.Items)
	}

	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return cfg, nil
}
