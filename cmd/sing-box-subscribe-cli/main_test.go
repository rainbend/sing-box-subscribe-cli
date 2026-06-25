package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	stdout, _, err := executeCommand("list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "sb-config-1.14.json") {
		t.Fatalf("list output missing sb-config-1.14.json:\n%s", stdout)
	}
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := executeCommand("version")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "dev\n" {
		t.Fatalf("stdout = %q, want dev version", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRootErrorPrintsHelpAndError(t *testing.T) {
	stdout, stderr, err := executeCommand()
	if err == nil {
		t.Fatal("expected missing subscription source error")
	}
	if err.Error() != "subscription URL or file is required" {
		t.Fatalf("error = %q, want missing subscription source", err)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage:\n  sing-box-sub [subscription URL or file]") {
		t.Fatalf("stderr missing usage:\n%s", stderr)
	}
	if !strings.Contains(stderr, `--template string`) {
		t.Fatalf("stderr missing flags:\n%s", stderr)
	}
	if !strings.Contains(stderr, "error: subscription URL or file is required") {
		t.Fatalf("stderr missing error:\n%s", stderr)
	}
}

func TestRootAcceptsSubscriptionAsPositionalArgument(t *testing.T) {
	dir := t.TempDir()
	subscriptionPath := filepath.Join(dir, "subscription.yaml")
	outputPath := filepath.Join(dir, "nodes.json")
	subscription := []byte(`
proxies:
  - { name: Japan, type: vmess, server: example.com, port: 443, uuid: 00000000-0000-0000-0000-000000000000 }
`)
	if err := os.WriteFile(subscriptionPath, subscription, 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := executeCommand(subscriptionPath, "--only-nodes", "--out", outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "generated 1 nodes") {
		t.Fatalf("stderr missing generated count:\n%s", stderr)
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `"tag": "Japan"`) {
		t.Fatalf("output missing node:\n%s", output)
	}
}

func TestRootRejectsDuplicateSubscriptionSources(t *testing.T) {
	_, _, err := executeCommand("positional.yaml", "--url", "flag.yaml")
	if err == nil {
		t.Fatal("expected duplicate subscription source error")
	}
	if err.Error() != "subscription source specified both as argument and --url" {
		t.Fatalf("error = %q, want duplicate subscription source", err)
	}
}

func TestTemplateDefaultsToSingBox114(t *testing.T) {
	cmd := newRootCommand()
	flag := cmd.Flags().Lookup("template")
	if flag == nil {
		t.Fatal("missing template flag")
	}
	if flag.DefValue != defaultTemplate {
		t.Fatalf("template default = %q, want %q", flag.DefValue, defaultTemplate)
	}
}

func executeCommand(args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := executeRootCommand(context.Background(), args, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}
