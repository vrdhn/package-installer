package downloader

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockTask struct {
	lastPercent int
	lastMsg     string
}

func (m *mockTask) Log(msg string)                      {}
func (m *mockTask) SetStage(name string, target string) {}
func (m *mockTask) Progress(percent int, message string) {
	m.lastPercent = percent
	m.lastMsg = message
}
func (m *mockTask) Done() {}

func TestHTTPDownload(t *testing.T) {
	content := []byte("some large content to test download")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer ts.Close()

	d := NewDefaultDownloader()
	buf := &bytes.Buffer{}
	task := &mockTask{}

	err := d.Download(context.Background(), ts.URL, buf, task)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("Content mismatch")
	}

	if task.lastPercent != 100 {
		t.Errorf("Expected 100%% progress, got %d", task.lastPercent)
	}
}

func TestHTTPRedirect(t *testing.T) {
	content := []byte("redirected content")

	// Target server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer ts.Close()

	// Redirect server
	rs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ts.URL, http.StatusMovedPermanently)
	}))
	defer rs.Close()

	d := NewDefaultDownloader()
	buf := &bytes.Buffer{}
	task := &mockTask{}

	err := d.Download(context.Background(), rs.URL, buf, task)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("Content mismatch, got %q", buf.String())
	}
}

func TestUnsupportedScheme(t *testing.T) {
	d := NewDefaultDownloader()
	err := d.Download(context.Background(), "ftp://example.com", &bytes.Buffer{}, &mockTask{})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("unsupported scheme")) {
		t.Errorf("Expected unsupported scheme error, got: %v", err)
	}
}
