package utils

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetBaseURL returns the base URL for the application
// If PUBLIC_URL is configured, use that; otherwise detect from request
func GetBaseURL(c *fiber.Ctx, publicURL string) string {
	// If PUBLIC_URL is configured and not default, use it
	if publicURL != "" && publicURL != "http://localhost:3000" {
		return publicURL
	}

	// Otherwise detect from request headers (for local development)
	scheme := "http"
	if c.Get("X-Forwarded-Proto") == "https" || c.Protocol() == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Get("Host"))
}

// GenerateFilename generates a filename based on expiry time and extension
func GenerateFilename(originalFilename string, expiryTime time.Time) string {
	// Get extension from original file
	originalExt := GetFileExtension(originalFilename)
	if originalExt == "" {
		originalExt = ".bin" // default extension if none
	}

	uuid := uuid.New().String()

	return fmt.Sprintf("%s_%d%s", uuid, expiryTime.Unix(), originalExt)
}

// GetFileExtension extracts file extension from filename
func GetFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}
	return ""
}

// ParseTimestampFromFilename extracts unix timestamp from filename
func ParseTimestampFromFilename(filename string) (int64, error) {
	// Remove extension to get unix timestamp
	filenameWithoutExt := filename
	if ext := GetFileExtension(filename); ext != "" {
		filenameWithoutExt = filename[:len(filename)-len(ext)]
	}

	// Parse as int64
	timestamp := int64(0)
	for _, char := range filenameWithoutExt {
		if char >= '0' && char <= '9' {
			timestamp = timestamp*10 + int64(char-'0')
		} else {
			return 0, fmt.Errorf("invalid timestamp format")
		}
	}

	return timestamp, nil
}

// IsFileExpired checks if a file has expired based on its filename
func IsFileExpired(filename string, currentTime time.Time) (bool, error) {
	timestamp, err := ParseTimestampFromFilename(filename)
	if err != nil {
		return false, err
	}

	expiryTime := time.Unix(timestamp, 0)
	return currentTime.After(expiryTime), nil
}
