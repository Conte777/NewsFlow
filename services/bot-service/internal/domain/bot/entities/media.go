// Package entities contains domain entities
package entities

// Media type constants
const (
	MediaTypePhoto     = "photo"
	MediaTypeVideo     = "video"
	MediaTypeVideoNote = "video_note"
	MediaTypeVoice     = "voice"
	MediaTypeAudio     = "audio"
	MediaTypeDocument  = "document"
)

// MediaMetadata contains media type and attributes for proper rendering
type MediaMetadata struct {
	Type     string // photo, video, video_note, voice, audio, document
	Width    int    // Video/VideoNote width
	Height   int    // Video/VideoNote height
	Duration int    // Video/VideoNote/Voice/Audio duration in seconds
}

// DownloadedFile represents a file downloaded from S3
// After first upload to Telegram, FileID is populated and Data can be cleared
type DownloadedFile struct {
	Data        []byte         // Raw file data (cleared after first upload)
	Filename    string         //
	ContentType string         //
	MediaType   string         // "photo", "video", "video_note", "voice", "audio", "document"
	FileID      string         // Telegram file_id (populated after first upload)
	Metadata    *MediaMetadata // Optional metadata with dimensions and duration
}
