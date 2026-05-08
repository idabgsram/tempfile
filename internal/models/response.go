package models

import "time"

// UploadResponse represents the response after successful file upload
type UploadResponse struct {
	Message      string    `json:"message"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	SizeHuman    string    `json:"size_human"`
	ExpiresAt    time.Time `json:"expires_at"`
	ExpiresIn    string    `json:"expires_in"`
	DownloadURL  string    `json:"download_url"`
}

// PasteResponse is returned by POST /paste in JSON mode.
type PasteResponse struct {
	Message   string    `json:"message"`
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	RawURL    string    `json:"raw_url"`
	Size      int64     `json:"size"`
	SizeHuman string    `json:"size_human"`
	ExpiresAt time.Time `json:"expires_at"`
	ExpiresIn string    `json:"expires_in"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
}

// WebPageData represents data passed to web templates
type WebPageData struct {
	Title            string
	Theme            string
	FileExpiryHours  int
	MaxFileSize      int64
	MaxFileSizeHuman string
	BaseURL          string
	Filename         string
	OriginalName     string
	Size             string
	SizeHuman        string
	ExpiresAt        string
	ExpiresIn        string
	ErrorTitle       string
	ErrorMessage     string
	ErrorDetail      string

	// Paste view fields
	PasteID      string
	PasteContent string
	PasteRawURL  string
	PasteURL     string
}
