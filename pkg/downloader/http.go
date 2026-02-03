package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"pi/pkg/display"
	"time"

	"github.com/dustin/go-humanize"
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

func (h *httpHandler) Download(ctx context.Context, uri string, w io.Writer, task display.Task) error {
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
		task:  task,
		total: size,
		start: time.Now(),
	}

	_, err = io.Copy(io.MultiWriter(w, pw), resp.Body)
	return err
}

// Mutable
type progressWriter struct {
	task    display.Task
	total   int64
	written int64
	start   time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	if pw.total > 0 {
		percent := int((float64(pw.written) / float64(pw.total)) * 100)
		elapsed := time.Since(pw.start).Seconds()
		speed := float64(pw.written) / elapsed
		msg := fmt.Sprintf("%s / %s (%s/s)",
			humanize.Bytes(uint64(pw.written)),
			humanize.Bytes(uint64(pw.total)),
			humanize.Bytes(uint64(speed)))
		pw.task.Progress(percent, msg)
	} else {
		pw.task.Progress(0, fmt.Sprintf("%s downloaded", humanize.Bytes(uint64(pw.written))))
	}

	return n, nil
}
