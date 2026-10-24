package attachment

import (
	"net/http"
	"strings"
)

func detectedImageMime(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	n := len(raw)
	if n > 512 {
		n = 512
	}
	mime := http.DetectContentType(raw[:n])
	if imageExt(mime) == "" {
		return ""
	}
	return mime
}

func imageExt(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}
	return ""
}
