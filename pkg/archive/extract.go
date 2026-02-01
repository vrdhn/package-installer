package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Extensions returns a list of file extensions that the archive module can extract
// for the given operating system.
func Extensions(os config.OSType) []string {
	switch os {
	case config.OSWindows:
		return []string{".zip"}
	case config.OSDarwin:
		return []string{".tar.gz", ".tar.zst", ".zip", ".tgz", ".tar"}
	default: // Linux and others
		return []string{".tar.gz", ".tar.zst", ".tgz", ".tar"}
	}
}

// SupportedExtensions returns a list of all file extensions that the archive module can extract.
func SupportedExtensions() []string {
	return []string{".zip", ".tar", ".tar.gz", ".tgz", ".tar.zst"}
}

// IsSupported returns true if the filename has a supported archive extension for the given OS.
func IsSupported(os config.OSType, filename string) bool {
	for _, ext := range Extensions(os) {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

// Extract extracts the contents of the archive at src into the directory dest.
// It supports .zip, .tar, .tar.gz, .tgz, and .tar.zst formats.
func Extract(src string, dest string) error {
	if strings.HasSuffix(src, ".zip") {
		return extractZip(src, dest)
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	var r io.Reader = f

	if strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tgz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		r = gzr
	} else if strings.HasSuffix(src, ".tar.zst") {
		zr, err := zstd.NewReader(f)
		if err != nil {
			return fmt.Errorf("failed to create zstd reader: %w", err)
		}
		defer zr.Close()
		r = zr
	} else if strings.HasSuffix(src, ".tar") {
		// Plain tar, reader is file
	} else {
		return fmt.Errorf("unsupported archive format: %s", src)
	}

	return extractTar(r, dest)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		err := extractFile(f.Name, f.FileInfo(), dest, func() (io.ReadCloser, error) {
			return f.Open()
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		err = extractFile(header.Name, header.FileInfo(), dest, func() (io.ReadCloser, error) {
			return io.NopCloser(tr), nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// extractFile is a helper to extract a single file/dir.
// opener is a function that returns a reader for the file content.
func extractFile(name string, info os.FileInfo, dest string, opener func() (io.ReadCloser, error)) error {
	// Secure path calculation (Zip Slip protection)
	target := filepath.Join(dest, name)
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path in archive: %s", name)
	}

	if info.IsDir() {
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", target, err)
		}
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
	}

	// Open destination
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", target, err)
	}
	defer f.Close()

	// Open source
	rc, err := opener()
	if err != nil {
		return fmt.Errorf("failed to open archive entry %s: %w", name, err)
	}
	// Note: For tar, rc is NopCloser(tr), so Close() does nothing, which is correct (we don't close tar stream)
	// For zip, rc is the file reader, which needs closing.
	defer rc.Close()

	_, err = io.Copy(f, rc)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", target, err)
	}
	return nil
}
