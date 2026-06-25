package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed *.json
var embeddedFS embed.FS

const userTemplateRelDir = ".config/sing-box-subscribe/internal/templates"

func List() ([]string, error) {
	entries, err := embeddedFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded templates: %w", err)
	}

	seen := make(map[string]struct{}, len(entries))
	addJSONEntries(seen, entries)

	if dir, ok := userTemplateDir(); ok {
		entries, err := os.ReadDir(dir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read user templates %q: %w", dir, err)
		}
		if err == nil {
			addJSONEntries(seen, entries)
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func Read(name string) ([]byte, bool, error) {
	normalized := NormalizeName(name)
	if normalized == "" {
		return nil, false, nil
	}

	for _, candidate := range candidateNames(normalized) {
		data, err := embeddedFS.ReadFile(candidate)
		if err == nil {
			return data, true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, false, err
		}
	}

	if dir, ok := userTemplateDir(); ok {
		for _, candidate := range candidateNames(normalized) {
			path := filepath.Join(dir, candidate)
			data, err := os.ReadFile(path)
			if err == nil {
				return data, true, nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, false, err
			}
		}
	}

	return nil, false, nil
}

func userTemplateDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}
	return filepath.Join(home, userTemplateRelDir), true
}

func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "embed:")
	name = strings.TrimPrefix(name, "embedded:")
	name = strings.TrimPrefix(name, "templates/")
	name = filepath.Base(name)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func addJSONEntries(names map[string]struct{}, entries []fs.DirEntry) {
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		names[entry.Name()] = struct{}{}
	}
}

func candidateNames(name string) []string {
	if strings.HasSuffix(name, ".json") {
		return []string{name}
	}
	return []string{name, name + ".json"}
}
