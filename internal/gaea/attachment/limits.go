package attachment

import (
	"encoding/base64"
	"errors"
)

const (
	// MaxFileBytes is the fixed safety limit for non-image attachments.
	MaxFileBytes int64 = 25 << 20
	// MaxImageBytes preserves the existing image limit.
	MaxImageBytes int64 = 10 << 20
)

var (
	ErrFileTooLarge  = errors.New("attachment exceeds 25 MiB")
	ErrImageTooLarge = errors.New("image attachment exceeds 10 MiB")
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
