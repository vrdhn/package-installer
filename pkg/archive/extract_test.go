package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestExtract(t *testing.T) {
	tempDir := t.TempDir()

	// Test data
	fileName := "test.txt"
	fileContent := "hello world"
	dirName := "subdir"
	subFileName := "sub.txt"
	subFileContent := "hello sub"

	// Helper to create valid archive content
	createContent := func(w func(name string, content []byte) error) error {
		if err := w(fileName, []byte(fileContent)); err != nil {
			return err
		}
		if err := w(filepath.Join(dirName, subFileName), []byte(subFileContent)); err != nil {
			return err
		}
		return nil
	}

	// 1. Test Zip
	zipPath := filepath.Join(tempDir, "test.zip")
	createZip(t, zipPath, createContent)
	testExtraction(t, zipPath, fileContent, subFileContent)

	// 2. Test Tar
	tarPath := filepath.Join(tempDir, "test.tar")
	createTar(t, tarPath, nil, createContent)
	testExtraction(t, tarPath, fileContent, subFileContent)

	// 3. Test Tar.gz
	tgzPath := filepath.Join(tempDir, "test.tar.gz")
	createTar(t, tgzPath, func(w io.Writer) io.WriteCloser {
		return gzip.NewWriter(w)
	}, createContent)
	testExtraction(t, tgzPath, fileContent, subFileContent)

	// 4. Test Tar.zst
	zstPath := filepath.Join(tempDir, "test.tar.zst")
	createTar(t, zstPath, func(w io.Writer) io.WriteCloser {
		e, _ := zstd.NewWriter(w)
		return e
	}, createContent)
	testExtraction(t, zstPath, fileContent, subFileContent)
}

func createZip(t *testing.T, path string, contentGen func(func(string, []byte) error) error) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	err = contentGen(func(name string, content []byte) error {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(content)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createTar(t *testing.T, path string, compressor func(io.Writer) io.WriteCloser, contentGen func(func(string, []byte) error) error) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var w io.WriteCloser = f
	if compressor != nil {
		w = compressor(f)
		defer w.Close()
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	err = contentGen(func(name string, content []byte) error {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err := tw.Write(content)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func testExtraction(t *testing.T, archivePath string, expectFile, expectSubFile string) {
	dest := filepath.Join(filepath.Dir(archivePath), "extract_"+filepath.Base(archivePath))
	err := Extract(archivePath, dest)
	if err != nil {
		t.Fatalf("Extract failed for %s: %v", archivePath, err)
	}

	checkFile(t, filepath.Join(dest, "test.txt"), expectFile)
	checkFile(t, filepath.Join(dest, "subdir", "sub.txt"), expectSubFile)
}

func checkFile(t *testing.T, path, content string) {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read extracted file %s: %v", path, err)
	}
	if string(b) != content {
		t.Errorf("File %s content mismatch. Want %q, got %q", path, content, string(b))
	}
}
