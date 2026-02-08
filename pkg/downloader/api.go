// Package downloader provides a modular system for retrieving remote resources.
// It supports multiple schemes (HTTP, HTTPS, etc.) and reports progress via the display package.
package downloader

import (
	"context"
	"io"
	"pi/pkg/display"
)

// Downloader manages the retrieval of resources from various URIs.
type Downloader interface {
	// Download retrieves the resource at the specified URI and writes it to w.
	// It uses the provided display Task to report progress and logs.
	Download(ctx context.Context, uri string, w io.Writer, task display.Task) error
}

// SchemeHandler defines the interface for handling specific URI schemes (e.g., "http://").
type SchemeHandler interface {
	// Download executes the download for a URI supported by this handler.
	Download(ctx context.Context, uri string, w io.Writer, task display.Task) error
	// Schemes returns the list of URI schemes (e.g., ["http", "https"]) this handler can process.
	Schemes() []string
}
