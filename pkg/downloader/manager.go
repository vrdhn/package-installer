package downloader

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"pi/pkg/display"
	"strings"
)

// Mutable
type manager struct {
	handlers map[string]SchemeHandler
}

func NewDefaultDownloader() Downloader {
	m := &manager{
		handlers: make(map[string]SchemeHandler),
	}
	h := NewHTTPHandler()
	m.Register(h)
	return m
}

func (m *manager) Register(h SchemeHandler) {
	for _, scheme := range h.Schemes() {
		m.handlers[scheme] = h
	}
}

func (m *manager) Download(ctx context.Context, uri string, w io.Writer, task display.Task) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid uri: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	handler, ok := m.handlers[scheme]
	if !ok {
		return fmt.Errorf("unsupported scheme: %s", scheme)
	}

	return handler.Download(ctx, uri, w, task)
}
