package config

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestSandboxReadRootsIncludeWriteRoots(t *testing.T) {
	workspace := t.TempDir()
	writeRoot := filepath.Join(t.TempDir(), "write")
	readRoot := filepath.Join(t.TempDir(), "read")
	cfg := &Config{Sandbox: SandboxConfig{
		WorkspaceRoot: workspace,
		AllowWrite:    []string{writeRoot},
		AllowRead:     []string{readRoot},
	}}

	if got, want := cfg.WriteRoots(), []string{workspace, writeRoot}; !reflect.DeepEqual(got, want) {
		t.Fatalf("WriteRoots() = %v, want %v", got, want)
	}
	if got, want := cfg.ReadRoots(), []string{workspace, writeRoot, readRoot}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadRoots() = %v, want %v", got, want)
	}
}

func TestSandboxReadRootsExpandVariables(t *testing.T) {
	t.Setenv("GAEA_ACCESS_ROOT", t.TempDir())
	cfg := &Config{Sandbox: SandboxConfig{
		WorkspaceRoot: t.TempDir(),
		AllowRead:     []string{"${GAEA_ACCESS_ROOT}"},
	}}
	got := cfg.ReadRoots()
	if len(got) != 2 || got[1] == "${GAEA_ACCESS_ROOT}" {
		t.Fatalf("ReadRoots() did not expand allow_read: %v", got)
	}
}

func TestSandboxAllowReadRendersAndRoundTrips(t *testing.T) {
	cfg := Default()
	cfg.Sandbox.AllowWrite = []string{filepath.Join(t.TempDir(), "write")}
	cfg.Sandbox.AllowRead = []string{filepath.Join(t.TempDir(), "read")}
	rendered := RenderTOML(cfg)

	var got Config
	if _, err := toml.Decode(rendered, &got); err != nil {
		t.Fatalf("rendered config does not parse: %v\n%s", err, rendered)
	}
	if !reflect.DeepEqual(got.Sandbox.AllowRead, cfg.Sandbox.AllowRead) {
		t.Fatalf("allow_read round trip = %v, want %v", got.Sandbox.AllowRead, cfg.Sandbox.AllowRead)
	}
	if !reflect.DeepEqual(got.Sandbox.AllowWrite, cfg.Sandbox.AllowWrite) {
		t.Fatalf("allow_write round trip = %v, want %v", got.Sandbox.AllowWrite, cfg.Sandbox.AllowWrite)
	}
}
