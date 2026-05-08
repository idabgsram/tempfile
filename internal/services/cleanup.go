package services

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pandeptwidyaop/tempfile/internal/config"
	"github.com/pandeptwidyaop/tempfile/internal/utils"
)

// CleanupService handles expired file cleanup
type CleanupService struct {
	config *config.Config
}

// NewCleanupService creates a new cleanup service instance
func NewCleanupService(cfg *config.Config) *CleanupService {
	return &CleanupService{
		config: cfg,
	}
}

// Start starts the cleanup routine
func (s *CleanupService) Start() {
	ticker := time.NewTicker(time.Duration(s.config.CleanupIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Printf("🧹 Cleanup routine started (interval: %d second(s))", s.config.CleanupIntervalSeconds)

	for range ticker.C {
		s.cleanupExpiredFiles()
	}
}

// cleanupExpiredFiles removes expired files from the upload directory and pastes subdirectory.
func (s *CleanupService) cleanupExpiredFiles() {
	now := time.Now()
	cleanedCount := 0

	cleanedCount += s.cleanupDir(s.config.UploadDir, now)
	cleanedCount += s.cleanupDir(filepath.Join(s.config.UploadDir, "pastes"), now)

	if cleanedCount > 0 {
		log.Printf("🗑️  Cleaned up %d expired file(s)", cleanedCount)
	}
}

// cleanupDir scans a single directory (non-recursive) and deletes expired files.
func (s *CleanupService) cleanupDir(dir string, now time.Time) int {
	files, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading directory %s: %v", dir, err)
		}
		return 0
	}

	cleaned := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()

		expired, err := utils.IsFileExpired(filename, now)
		if err != nil {
			continue
		}
		if !expired {
			continue
		}

		filePath := filepath.Join(dir, filename)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error removing expired file %s: %v", filePath, err)
			continue
		}
		cleaned++
		if s.config.Debug {
			log.Printf("Removed expired file: %s", filePath)
		}
	}
	return cleaned
}
