package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type userEnsurer struct {
	client *http.Client
	logger *zap.Logger
	mu     sync.Mutex
	seen   map[string]bool
}

func newUserEnsurer(socket string, logger *zap.Logger) *userEnsurer {
	return &userEnsurer{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socket)
				},
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger: logger,
		seen:   map[string]bool{},
	}
}

func (e *userEnsurer) ensure(username string) {
	e.mu.Lock()
	done := e.seen[username]
	e.mu.Unlock()
	if done {
		return
	}

	req, err := http.NewRequest(http.MethodGet, "http://navidrome/app/", nil)
	if err != nil {
		return
	}
	req.Header.Set(userHeader, username)
	resp, err := e.client.Do(req)
	if err != nil {
		e.logger.Warn("ensure navidrome user", zap.String("user", username), zap.Error(err))
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < http.StatusInternalServerError {
		e.mu.Lock()
		e.seen[username] = true
		e.mu.Unlock()
	}
}
