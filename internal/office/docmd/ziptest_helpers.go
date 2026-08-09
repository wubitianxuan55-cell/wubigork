package docmd

import (
	"archive/zip"
	"io"
)

// zipWrite writes map[zipPath]content as a zip archive (test helper).
func zipWrite(w io.Writer, entries map[string]string) error {
	zw := zip.NewWriter(w)
	for name, content := range entries {
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			return err
		}
	}
	return zw.Close()
}
