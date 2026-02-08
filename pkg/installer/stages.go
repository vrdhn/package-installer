package installer

import (
	"context"
	"fmt"
	"os"
	"pi/pkg/archive"
	"pi/pkg/cache"
	"pi/pkg/display"
	"pi/pkg/downloader"
)

// DownloadStage retrieves the package archive from the remote URL.
// It uses the local cache to avoid redundant downloads if the file already exists.
func DownloadStage(ctx context.Context, plan *Plan, task display.Task) error {
	task.SetStage("Download", plan.DownloadPath)
	d := downloader.NewDefaultDownloader()

	err := cache.Ensure(plan.DownloadPath, func() error {
		task.Log(fmt.Sprintf("Downloading %s", plan.Package.URL))
		f, err := os.Create(plan.DownloadPath)
		if err != nil {
			return err
		}
		defer f.Close()

		return d.Download(ctx, plan.Package.URL, f, task)
	})

	return err
}

// ExtractStage unpacks the package archive into the designated installation directory.
// It uses a temporary directory during extraction to ensure the final installation is atomic.
func ExtractStage(ctx context.Context, plan *Plan, task display.Task) error {
	task.SetStage("Extract", plan.InstallPath)
	err := cache.Ensure(plan.InstallPath, func() error {
		task.Log(fmt.Sprintf("Extracting to %s", plan.InstallPath))

		// Create temporary directory for extraction to ensure atomicity
		// If extraction fails, we don't want a half-extracted directory at InstallPath
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

		// After extraction, move to final path
		// We might need to handle the case where the archive contains a single top-level folder
		// which is common for nodejs/java distributions.
		// For now, let's just move the tmpDir to InstallPath.
		return os.Rename(tmpDir, plan.InstallPath)
	})

	return err
}

// Install coordinates the full installation process, including downloading and extracting
// the package. It skips these steps if the package is already present on disk.
func Install(ctx context.Context, plan *Plan, task display.Task) error {
	// 0. Check if already installed
	if _, err := os.Stat(plan.InstallPath); err == nil {
		task.SetStage("Done", plan.InstallPath)
		task.Progress(100, "Already installed")
		return nil
	}

	if err := DownloadStage(ctx, plan, task); err != nil {
		return fmt.Errorf("download stage failed: %w", err)
	}

	// Reset progress for next stage
	task.Progress(0, "Preparing to extract")

	if err := ExtractStage(ctx, plan, task); err != nil {
		return fmt.Errorf("extract stage failed: %w", err)
	}

	task.Progress(100, "Installation complete")
	return nil
}
