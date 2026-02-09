// Package downloader provides a modular system for retrieving remote resources.
package downloader

import (
	"context"
	"io"
)

// Downloader manages the retrieval of resources from various URIs.
type Downloader = *manager

// SchemeHandler defines the interface for handling specific URI schemes (e.g., "http://").
type SchemeHandler interface {
	// Download executes the download for a URI supported by this handler.
	Download(ctx context.Context, uri string, w io.Writer) error
	// Schemes returns the list of URI schemes (e.g., ["http", "https"]) this handler can process.
	Schemes() []string
}
