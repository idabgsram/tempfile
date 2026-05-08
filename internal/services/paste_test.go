package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pandeptwidyaop/tempfile/internal/config"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		UploadDir:       dir,
		MaxFileSize:     1024,
		FileExpiryHours: 1,
	}
}

func TestPasteService_Create_WritesFileUnderPastesDir(t *testing.T) {
	cfg := newTestConfig(t)
	svc := NewPasteService(cfg)

	res, err := svc.Create("hello world")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if res.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	matches, _ := filepath.Glob(filepath.Join(cfg.UploadDir, "pastes", res.ID+"_*.txt"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 file matching id, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("file content mismatch: %q", string(data))
	}
}

func TestPasteService_Create_RejectsEmpty(t *testing.T) {
	svc := NewPasteService(newTestConfig(t))
	if _, err := svc.Create("   \n\t "); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestPasteService_Create_RejectsOversize(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.MaxFileSize = 4
	svc := NewPasteService(cfg)
	if _, err := svc.Create("more than four bytes"); err == nil {
		t.Fatal("expected error for oversize content")
	}
}

func TestPasteService_Get_ReturnsContent(t *testing.T) {
	cfg := newTestConfig(t)
	svc := NewPasteService(cfg)
	created, err := svc.Create("the body")
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Content != "the body" {
		t.Fatalf("content mismatch: %q", got.Content)
	}
	if got.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero ExpiresAt")
	}
}

func TestPasteService_Get_NotFound(t *testing.T) {
	svc := NewPasteService(newTestConfig(t))
	if _, err := svc.Get("nope"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestPasteService_Get_RejectsTraversal(t *testing.T) {
	svc := NewPasteService(newTestConfig(t))
	for _, id := range []string{"../etc/passwd", "..", "a/b", "a\\b"} {
		if _, err := svc.Get(id); err == nil {
			t.Fatalf("expected error for unsafe id %q", id)
		}
	}
}
