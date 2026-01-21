// Package entities contains domain entities
package entities

// DownloadedFile represents a file downloaded from S3
// After first upload to Telegram, FileID is populated and Data can be cleared
type DownloadedFile struct {
	Data        []byte // Raw file data (cleared after first upload)
	Filename    string
	ContentType string
	MediaType   string // "photo", "video", "document"
	FileID      string // Telegram file_id (populated after first upload)
}
