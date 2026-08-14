package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	dest := "D:\stage"
	for _, name := range []string{
		"C:/evil.txt",
		"C:evil.txt",
		"C:\evil.txt",
		"\\server\share\evil.txt",
		"//server/share/evil.txt",
		"..\..\evil.txt",
		"a/../../b.txt",
		"./x/../..",
	} {
		rel, err := filepath.Rel(dest, name)
		_ = rel
		joined := filepath.Join(dest, filepath.FromSlash(name))
		fmt.Printf("name=%-28q clean=%-28q join=%-28q err=%v
", name, filepath.Clean(filepath.FromSlash(name)), joined, err)
	}
}
