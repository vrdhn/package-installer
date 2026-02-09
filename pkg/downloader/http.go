package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Immutable
type httpHandler struct {
	client *http.Client
}

func NewHTTPHandler() SchemeHandler {
	return &httpHandler{
		client: &http.Client{
			Timeout: 0, // Handled by context
		},
	}
}

func (h *httpHandler) Schemes() []string {
	return []string{"http", "https"}
}

func (h *httpHandler) Download(ctx context.Context, uri string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	size := resp.ContentLength

	pw := &progressWriter{
		total: size,
		start: time.Now(),
		uri:   uri,
	}

	_, err = io.Copy(io.MultiWriter(w, pw), resp.Body)
	return err
}

// Mutable
type progressWriter struct {
	total   int64
	written int64
	start   time.Time
	uri     string
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	if pw.total > 0 {
		elapsed := time.Since(pw.start).Seconds()
		if elapsed > 1 {
			// Log progress if needed
		}
	}

	return n, nil
}
