package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListIncludesBundledTemplates(t *testing.T) {
	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	want := "sb-config-1.14.json"
	for _, name := range names {
		if name == want {
			return
		}
	}
	t.Fatalf("template %q not found in %v", want, names)
}

func TestReadTemplateByName(t *testing.T) {
	data, ok, err := Read("sb-config-1.14.json")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("template not found")
	}
	if len(data) < 1000 {
		t.Fatalf("template content too small: %d bytes", len(data))
	}
}

func TestListIncludesUserTemplates(t *testing.T) {
	writeUserTemplate(t, "custom.json", `{"outbounds":[]}`)

	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name == "custom.json" {
			return
		}
	}
	t.Fatalf("user template not found in %v", names)
}

func TestReadUserTemplateByName(t *testing.T) {
	want := `{"outbounds":[{"type":"direct","tag":"direct"}]}`
	writeUserTemplate(t, "custom.json", want)

	data, ok, err := Read("custom")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("template not found")
	}
	if strings.TrimSpace(string(data)) != want {
		t.Fatalf("template content = %q, want %q", data, want)
	}
}

func TestReadPrefersBundledTemplate(t *testing.T) {
	writeUserTemplate(t, "sb-config-1.14.json", `{}`)

	data, ok, err := Read("sb-config-1.14.json")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("template not found")
	}
	if len(data) < 1000 {
		t.Fatalf("expected bundled template, got %d bytes", len(data))
	}
}

func writeUserTemplate(t *testing.T, name, content string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, userTemplateRelDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
