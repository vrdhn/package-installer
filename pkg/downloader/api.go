package downloader

import (
	"context"
	"io"
	"pi/pkg/display"
)

// Downloader manages downloads for various URI schemes.
type Downloader interface {
	// Download downloads the resource at uri to the writer w.
	// It uses the provided display to report progress.
	// The caller is responsible for calling task.Done().
	Download(ctx context.Context, uri string, w io.Writer, task display.Task) error
}

// SchemeHandler is the interface for handling a specific URI scheme (e.g., http, git).
type SchemeHandler interface {
	// Download handles the download for the given uri.
	Download(ctx context.Context, uri string, w io.Writer, task display.Task) error
	// Schemes returns the list of schemes this handler supports.
	Schemes() []string
}
