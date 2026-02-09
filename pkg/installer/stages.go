package installer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"pi/pkg/archive"
	"pi/pkg/cache"
	"pi/pkg/downloader"
)

// DownloadStage retrieves the package archive from the remote URL.
// It uses the local cache to avoid redundant downloads if the file already exists.
func DownloadStage(ctx context.Context, plan *Plan) error {
	slog.Info("Downloading package", "url", plan.Package.URL, "path", plan.DownloadPath)
	d := downloader.NewDefaultDownloader()

	err := cache.Ensure(plan.DownloadPath, func() error {
		f, err := os.Create(plan.DownloadPath)
		if err != nil {
			return err
		}
		defer f.Close()

		return d.Download(ctx, plan.Package.URL, f)
	})

	return err
}

// ExtractStage unpacks the package archive into the designated installation directory.
// It uses a temporary directory during extraction to ensure the final installation is atomic.
func ExtractStage(ctx context.Context, plan *Plan) error {
	slog.Info("Extracting package", "path", plan.InstallPath)
	err := cache.Ensure(plan.InstallPath, func() error {
		// Create temporary directory for extraction to ensure atomicity
		tmpDir := plan.InstallPath + ".tmp"
		if err := os.RemoveAll(tmpDir); err != nil {
			return err
		}
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		if err := archive.Extract(plan.DownloadPath, tmpDir); err != nil {
			return err
		}

		return os.Rename(tmpDir, plan.InstallPath)
	})

	return err
}

// Install coordinates the full installation process, including downloading and extracting
// the package. It skips these steps if the package is already present on disk.
func Install(ctx context.Context, plan *Plan) error {
	// 0. Check if already installed
	if _, err := os.Stat(plan.InstallPath); err == nil {
		slog.Debug("Package already installed", "path", plan.InstallPath)
		return nil
	}

	if err := DownloadStage(ctx, plan); err != nil {
		return fmt.Errorf("download stage failed: %w", err)
	}

	if err := ExtractStage(ctx, plan); err != nil {
		return fmt.Errorf("extract stage failed: %w", err)
	}

	slog.Info("Installation complete", "package", plan.Package.Name, "version", plan.Package.Version)
	return nil
}
