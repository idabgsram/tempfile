package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pandeptwidyaop/tempfile/internal/config"
)

const pastesSubdir = "pastes"

// PasteService manages paste creation and retrieval on disk.
type PasteService struct {
	config *config.Config
}

// NewPasteService creates a new PasteService.
func NewPasteService(cfg *config.Config) *PasteService {
	return &PasteService{config: cfg}
}

// CreatedPaste is returned from Create.
type CreatedPaste struct {
	ID        string
	Filename  string
	Size      int64
	ExpiresAt time.Time
}

// FetchedPaste is returned from Get.
type FetchedPaste struct {
	ID        string
	Content   string
	Size      int64
	ExpiresAt time.Time
}

var (
	ErrPasteNotFound  = errors.New("paste not found")
	ErrInvalidPasteID = errors.New("invalid paste id")
)

// Create writes content to UploadDir/pastes/{id}_{unix_expiry}.txt.
func (s *PasteService) Create(content string) (*CreatedPaste, error) {
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("content cannot be empty")
	}
	if int64(len(content)) > s.config.MaxFileSize {
		return nil, fmt.Errorf("paste size exceeds limit of %d bytes", s.config.MaxFileSize)
	}

	dir := filepath.Join(s.config.UploadDir, pastesSubdir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create pastes dir: %w", err)
	}

	id := uuid.New().String()
	expiry := time.Now().Add(time.Duration(s.config.FileExpiryHours) * time.Hour)
	filename := fmt.Sprintf("%s_%d.txt", id, expiry.Unix())
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		return nil, fmt.Errorf("write paste: %w", err)
	}

	return &CreatedPaste{
		ID:        id,
		Filename:  filename,
		Size:      int64(len(content)),
		ExpiresAt: expiry,
	}, nil
}

// Get reads the paste with the given id.
func (s *PasteService) Get(id string) (*FetchedPaste, error) {
	if !isSafePasteID(id) {
		return nil, ErrInvalidPasteID
	}

	dir := filepath.Join(s.config.UploadDir, pastesSubdir)
	matches, err := filepath.Glob(filepath.Join(dir, id+"_*.txt"))
	if err != nil {
		return nil, fmt.Errorf("glob paste: %w", err)
	}
	if len(matches) == 0 {
		return nil, ErrPasteNotFound
	}

	path := matches[0]
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPasteNotFound
		}
		return nil, fmt.Errorf("read paste: %w", err)
	}

	expiresAt, err := parseExpiryFromPasteFilename(filepath.Base(path))
	if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		_ = os.Remove(path)
		return nil, ErrPasteNotFound
	}

	return &FetchedPaste{
		ID:        id,
		Content:   string(data),
		Size:      int64(len(data)),
		ExpiresAt: expiresAt,
	}, nil
}

// PastesDir returns the absolute path to the pastes subdirectory.
func (s *PasteService) PastesDir() string {
	return filepath.Join(s.config.UploadDir, pastesSubdir)
}

func isSafePasteID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func parseExpiryFromPasteFilename(name string) (time.Time, error) {
	base := strings.TrimSuffix(name, ".txt")
	idx := strings.LastIndex(base, "_")
	if idx < 0 {
		return time.Time{}, fmt.Errorf("invalid paste filename: %s", name)
	}
	tsPart := base[idx+1:]
	var ts int64
	for _, c := range tsPart {
		if c < '0' || c > '9' {
			return time.Time{}, fmt.Errorf("invalid timestamp in: %s", name)
		}
		ts = ts*10 + int64(c-'0')
	}
	return time.Unix(ts, 0), nil
}
