package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOpenAPIYAMLPath_fromEnv(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(p, []byte("openapi: 3.1.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAPI_SPEC_PATH", p)
	got, err := ResolveOpenAPIYAMLPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(p) {
		t.Fatalf("got %q want %q", got, p)
	}
}
