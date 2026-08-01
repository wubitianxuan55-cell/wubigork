package builtin

import (
	"os"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

// ── Encoding-aware file I/O ────────────────────────────────────────────

// readFileEncoded reads a file and decodes its encoding to UTF-8.
func readFileEncoded(path string) (content string, enc fileenc.Kind, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	enc, _ = fileenc.Detect(b)
	return string(fileenc.Decode(b, enc)), enc, nil
}

// writeFileEncoded encodes content back to the given encoding and writes it.
func writeFileEncoded(path string, content string, enc fileenc.Kind, perm os.FileMode) error {
	return os.WriteFile(path, fileenc.Encode(content, enc), perm)
}
