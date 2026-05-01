package docs

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveOpenAPIYAMLPath devolve o primeiro caminho existente para o spec OpenAPI.
// Ordem: OPENAPI_SPEC_PATH → cwd api/openapi.yaml → exe-dir api/openapi.yaml.
func ResolveOpenAPIYAMLPath() (string, error) {
	if p := os.Getenv("OPENAPI_SPEC_PATH"); p != "" {
		if fileExists(p) {
			return filepath.Clean(p), nil
		}
		return "", fmt.Errorf("OPENAPI_SPEC_PATH file not found: %s", p)
	}

	candidates := []string{
		filepath.Join("api", "openapi.yaml"),
		filepath.Join("backend", "api", "openapi.yaml"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append([]string{
			filepath.Join(wd, "api", "openapi.yaml"),
			filepath.Join(wd, "backend", "api", "openapi.yaml"),
		}, candidates...)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "api", "openapi.yaml"),
		)
	}

	for _, p := range candidates {
		if fileExists(p) {
			return filepath.Clean(p), nil
		}
	}
	return "", fmt.Errorf("openapi.yaml not found (run API with cwd=backend/ ou defina OPENAPI_SPEC_PATH)")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
