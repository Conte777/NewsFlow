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
	PhotoSizeSourceLegacy           = 0
	PhotoSizeSourceThumbnail        = 1
	PhotoSizeSourceDialogPhotoBig   = 2
	PhotoSizeSourceDialogPhotoSmall = 3
	PhotoSizeSourceStickerSetThumb  = 4
	PhotoSizeSourceFullLegacy       = 5
)

// File ID encoding flags (from tg-file-decoder)
const (
	FileReferenceFlag = 1 << 25 // 0x02000000
	WebLocationFlag   = 1 << 24 // 0x01000000
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
// Based on specification from github.com/danog/tg-file-decoder
func encodeFileID(data FileIDData) string {
	buf := make([]byte, 0, 64)

	// Version constants (from current Telegram clients)
	const version = 4
	const subVersion = 47

	// 1. TypeID (4 bytes, LE) - includes FileReferenceFlag if present
	typeID := int32(data.Type)
	if len(data.FileReference) > 0 {
		typeID |= FileReferenceFlag
	}
	buf = appendInt32(buf, typeID)

	// 2. DCID (4 bytes, LE)
	buf = appendInt32(buf, int32(data.DCID))

	// 3. FileReference (if present) - TL-style bytes encoding
	if len(data.FileReference) > 0 {
		buf = appendTLBytes(buf, data.FileReference)
	}

	// 4. ID (8 bytes, LE)
	buf = appendInt64(buf, data.ID)

	// 5. AccessHash (8 bytes, LE)
	buf = appendInt64(buf, data.AccessHash)

	// 6. For Photo: PhotoSizeSource data
	if data.Type == TypePhoto {
		// SizeSource type (4 bytes, LE)
		buf = appendInt32(buf, int32(data.PhotoSizeSource))

		switch data.PhotoSizeSource {
		case PhotoSizeSourceThumbnail:
			// FileType (4 bytes, LE) - same as TypePhoto
			buf = appendInt32(buf, int32(TypePhoto))
			// ThumbType as 4-byte padded char (e.g., 'y' -> 0x79000000)
			buf = appendInt32(buf, int32(data.ThumbnailType))
		case PhotoSizeSourceFullLegacy:
			// VolumeID (8 bytes) + LocalID (4 bytes) - deprecated format
			buf = appendInt64(buf, data.VolumeID)
			buf = appendInt32(buf, data.LocalID)
		}
	}

	// 7. SubVersion + Version at the end (1 byte each)
	buf = append(buf, byte(subVersion))
	buf = append(buf, byte(version))

	// Apply RLE encoding and Base64URL encode
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

// appendTLBytes encodes bytes in Telegram TL-style format:
// - Length < 254: 1 byte length + data + padding to 4-byte boundary
// - Length >= 254: 0xFE + 3 bytes length (LE) + data + padding
func appendTLBytes(buf []byte, data []byte) []byte {
	length := len(data)

	if length < 254 {
		// Short encoding: 1 byte length
		buf = append(buf, byte(length))
		buf = append(buf, data...)
		// Padding to 4-byte boundary (length byte + data)
		paddingNeeded := (4 - (1+length)%4) % 4
		for i := 0; i < paddingNeeded; i++ {
			buf = append(buf, 0)
		}
	} else {
		// Long encoding: 0xFE marker + 3-byte length
		buf = append(buf, 0xFE)
		buf = append(buf, byte(length&0xFF))
		buf = append(buf, byte((length>>8)&0xFF))
		buf = append(buf, byte((length>>16)&0xFF))
		buf = append(buf, data...)
		// Padding to 4-byte boundary (data only, marker+length already 4 bytes)
		paddingNeeded := (4 - length%4) % 4
		for i := 0; i < paddingNeeded; i++ {
			buf = append(buf, 0)
		}
	}

	return buf
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
