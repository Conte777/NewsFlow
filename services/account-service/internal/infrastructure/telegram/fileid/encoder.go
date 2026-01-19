// Package fileid provides encoding of MTProto file data to Bot API file_id format.
// Based on the Telegram Bot API file_id specification.
// Reference: https://github.com/danog/tg-file-decoder
package fileid

import (
	"encoding/base64"
	"encoding/binary"

	"github.com/gotd/td/tg"
)

// FileType constants for Bot API (from tg-file-decoder)
const (
	TypeThumbnail        = 0
	TypeProfilePhoto     = 1
	TypePhoto            = 2
	TypeVoice            = 3
	TypeVideo            = 4
	TypeDocument         = 5
	TypeEncrypted        = 6
	TypeTemp             = 7
	TypeSticker          = 8
	TypeAudio            = 9
	TypeAnimation        = 10
	TypeEncryptedThumb   = 11
	TypeWallpaper        = 12
	TypeVideoNote        = 13
	TypeSecureRaw        = 14
	TypeSecure           = 15
	TypeBackground       = 16
	TypeDocumentAsFile   = 17
	TypeSize             = 18
	TypePhotoSizeSource  = 19
)

// PhotoSizeSource constants
const (
	PhotoSizeSourceLegacy          = 0
	PhotoSizeSourceThumbnail       = 1
	PhotoSizeSourceDialogPhotoBig  = 2
	PhotoSizeSourceDialogPhotoSmall = 3
	PhotoSizeSourceStickerSetThumb = 4
	PhotoSizeSourceFullLegacy      = 5
)

// FileIDData holds all data needed to encode a file_id
type FileIDData struct {
	Type          int
	DCID          int
	ID            int64
	AccessHash    int64
	FileReference []byte
	// Photo-specific fields
	PhotoSizeSource int
	VolumeID        int64
	LocalID         int32
	// Thumbnail-specific
	ThumbnailType byte
}

// EncodePhotoFileID converts tg.Photo to Bot API file_id
// Returns the largest available photo size encoded as file_id
func EncodePhotoFileID(photo *tg.Photo) string {
	if photo == nil {
		return ""
	}

	// Find the best (largest) photo size
	var bestSize *tg.PhotoSize
	var maxArea int

	for _, size := range photo.Sizes {
		if ps, ok := size.(*tg.PhotoSize); ok {
			area := ps.W * ps.H
			if area > maxArea {
				maxArea = area
				bestSize = ps
			}
		}
	}

	// Also check PhotoCachedSize for thumbnails
	if bestSize == nil {
		for _, size := range photo.Sizes {
			if cached, ok := size.(*tg.PhotoCachedSize); ok {
				bestSize = &tg.PhotoSize{
					Type: cached.Type,
					W:    cached.W,
					H:    cached.H,
				}
				break
			}
		}
	}

	thumbType := byte('y') // Default to 'y' (largest)
	if bestSize != nil && len(bestSize.Type) > 0 {
		thumbType = bestSize.Type[0]
	}

	data := FileIDData{
		Type:            TypePhoto,
		DCID:            photo.DCID,
		ID:              photo.ID,
		AccessHash:      photo.AccessHash,
		FileReference:   photo.FileReference,
		PhotoSizeSource: PhotoSizeSourceThumbnail,
		ThumbnailType:   thumbType,
	}

	return encodeFileID(data)
}

// EncodeDocumentFileID converts tg.Document to Bot API file_id
// fileType should be one of: TypeDocument, TypeVideo, TypeAudio, TypeVoice,
// TypeAnimation, TypeSticker, TypeVideoNote
func EncodeDocumentFileID(doc *tg.Document, fileType int) string {
	if doc == nil {
		return ""
	}

	// Use TypeDocument as default if invalid type provided
	if fileType < TypeDocument {
		fileType = TypeDocument
	}

	data := FileIDData{
		Type:          fileType,
		DCID:          doc.DCID,
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	return encodeFileID(data)
}

// DetectDocumentType determines the file type from document attributes
func DetectDocumentType(doc *tg.Document) int {
	if doc == nil {
		return TypeDocument
	}

	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeVideo:
			if a.RoundMessage {
				return TypeVideoNote
			}
			return TypeVideo
		case *tg.DocumentAttributeAudio:
			if a.Voice {
				return TypeVoice
			}
			return TypeAudio
		case *tg.DocumentAttributeSticker:
			return TypeSticker
		case *tg.DocumentAttributeAnimated:
			return TypeAnimation
		}
	}

	return TypeDocument
}

// encodeFileID serializes FileIDData to Bot API file_id format
func encodeFileID(data FileIDData) string {
	buf := make([]byte, 0, 64)

	// Version and subversion
	const version = 4
	const subVersion = 47

	buf = append(buf, byte(data.Type))
	buf = append(buf, byte(data.DCID))

	// File reference flag (1 if present)
	hasFileRef := byte(0)
	if len(data.FileReference) > 0 {
		hasFileRef = 1
	}
	buf = append(buf, hasFileRef)

	// For photos, add photo-specific data
	if data.Type == TypePhoto {
		// Photo size source
		buf = append(buf, byte(data.PhotoSizeSource))

		switch data.PhotoSizeSource {
		case PhotoSizeSourceThumbnail:
			// Thumbnail type (single char like 'y', 's', 'm', 'x', etc.)
			buf = append(buf, data.ThumbnailType)
		case PhotoSizeSourceFullLegacy:
			// Volume ID and local ID (deprecated, use 0)
			buf = appendInt64(buf, data.VolumeID)
			buf = appendInt32(buf, data.LocalID)
		}
	}

	// ID and AccessHash (always present)
	buf = appendInt64(buf, data.ID)
	buf = appendInt64(buf, data.AccessHash)

	// File reference (if present)
	if len(data.FileReference) > 0 {
		buf = appendVarint(buf, len(data.FileReference))
		buf = append(buf, data.FileReference...)
	}

	// Add version info at the end
	buf = append(buf, byte(subVersion))
	buf = append(buf, byte(version))

	// Encode to base64 with Telegram's URL-safe format
	return rleEncode(buf)
}

// appendInt64 appends a little-endian int64 to buffer
func appendInt64(buf []byte, v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return append(buf, b...)
}

// appendInt32 appends a little-endian int32 to buffer
func appendInt32(buf []byte, v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return append(buf, b...)
}

// appendVarint appends a variable-length integer
func appendVarint(buf []byte, v int) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(b, int64(v))
	return append(buf, b[:n]...)
}

// rleEncode encodes binary data to Telegram's base64 format with RLE for zeros
func rleEncode(data []byte) string {
	// Simple RLE encoding for consecutive zeros
	var encoded []byte
	i := 0
	for i < len(data) {
		if data[i] == 0 {
			// Count consecutive zeros
			count := 0
			for i < len(data) && data[i] == 0 && count < 255 {
				count++
				i++
			}
			encoded = append(encoded, 0, byte(count))
		} else {
			encoded = append(encoded, data[i])
			i++
		}
	}

	// Base64 encode with URL-safe alphabet
	result := base64.URLEncoding.EncodeToString(encoded)

	// Telegram uses base64 without padding
	for len(result) > 0 && result[len(result)-1] == '=' {
		result = result[:len(result)-1]
	}

	return result
}
