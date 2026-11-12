package attachment

import (
	"encoding/base64"
	"errors"
)

const (
	// MaxFileBytes is the fixed safety limit for non-image attachments
	// uploaded in a single bridge call (whole payload as one base64 string).
	MaxFileBytes int64 = 25 << 20
	// MaxChunkedFileBytes caps the assembled size of chunked uploads
	// (BeginChunk/AppendChunk/FinishChunk). Chunked uploads stream to disk
	// chunk by chunk, so the practical ceiling is disk space, not bridge
	// memory; 1 GiB stays firmly on the "document/CAD drawing" side.
	MaxChunkedFileBytes int64 = 1 << 30
	// MaxImageBytes preserves the existing image limit.
	MaxImageBytes int64 = 10 << 20
)

var (
	ErrFileTooLarge  = errors.New("attachment exceeds 25 MiB")
	ErrImageTooLarge = errors.New("image attachment exceeds 10 MiB")
	// ErrChunkedFileTooLarge is returned by AppendChunk when the running
	// total would exceed MaxChunkedFileBytes.
	ErrChunkedFileTooLarge = errors.New("chunked attachment exceeds 1 GiB")
)

// RejectEncodedFileBeforeDecode performs an allocation-free size check against
// the base64 payload length before DecodeString is called.
func RejectEncodedFileBeforeDecode(encoded string) error {
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > MaxFileBytes {
		return ErrFileTooLarge
	}
	return nil
}

func rejectEncodedImageBeforeDecode(encoded string) error {
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > MaxImageBytes {
		return ErrImageTooLarge
	}
	return nil
}
