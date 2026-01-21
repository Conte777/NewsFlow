// Package entities contains domain entities
package entities

// DownloadedFile represents a file downloaded from S3
type DownloadedFile struct {
	Data        []byte
	Filename    string
	ContentType string
	MediaType   string // "photo", "video", "document"
}
