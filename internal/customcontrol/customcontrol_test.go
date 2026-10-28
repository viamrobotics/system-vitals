package customcontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testModel = "acme:weather:station"
	testRel   = "ui/dist/index.html"
)

func setup(t *testing.T, manifest, html string) string {
	t.Helper()
	ClearCache()
	t.Cleanup(ClearCache)
	dir := t.TempDir()
	meta := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(meta, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if html != "" {
		p := filepath.Join(dir, filepath.FromSlash(testRel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(html), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return meta
}

const goodManifest = `{"models":[{"model":"acme:weather:station","custom_control":"ui/dist/index.html"},{"model":"acme:weather:bare"}]}`

func TestLoadHTMLAndRevision(t *testing.T) {
	meta := setup(t, goodManifest, "<h1>hi</h1>")
	a, err := Load(meta, testModel)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("<h1>hi</h1>"))
	if a.HTML != "<h1>hi</h1>" || a.Revision != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected artifact %+v", a)
	}
}

func TestLoadResolvesRelativeToManifestDir(t *testing.T) {
	meta := setup(t, goodManifest, "<p>x</p>")
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil { // differs from manifest directory
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if _, err := Load(meta, testModel); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCaches(t *testing.T) {
	meta := setup(t, goodManifest, "one")
	first, err := Load(meta, testModel)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(filepath.Dir(meta), filepath.FromSlash(testRel))
	if err := os.WriteFile(p, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, _ := Load(meta, testModel)
	if second != first {
		t.Fatalf("expected cached %+v, got %+v", first, second)
	}
	ClearCache()
	third, _ := Load(meta, testModel)
	if third.HTML != "two" || third.Revision == first.Revision {
		t.Fatalf("expected fresh artifact, got %+v", third)
	}
}

func TestHandle(t *testing.T) {
	meta := setup(t, goodManifest, "body")
	if resp, handled, err := Handle(map[string]any{"command": "other"}, meta, testModel); handled || resp != nil || err != nil {
		t.Fatalf("other command: %v %v %v", resp, handled, err)
	}
	if _, handled, _ := Handle(map[string]any{}, meta, testModel); handled {
		t.Fatal("empty command should not be handled")
	}
	resp, handled, err := Handle(map[string]any{"command": Command}, meta, testModel)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	sum := sha256.Sum256([]byte("body"))
	if resp["html"] != "body" || resp["revision"] != hex.EncodeToString(sum[:]) || resp["path"] != "ui/dist/index.html" {
		t.Fatalf("unexpected response %v", resp)
	}
}

func TestErrors(t *testing.T) {
	cmd := map[string]any{"command": Command}
	tests := []struct {
		name     string
		manifest string
		html     string
		model    string
		want     error
		mention  string
	}{
		{"no entry", goodManifest, "x", "acme:weather:nope", ErrNoEntry, "acme:weather:nope"},
		{"no control", goodManifest, "x", "acme:weather:bare", ErrNoControl, "acme:weather:bare"},
		{"missing file", goodManifest, "", testModel, ErrMissingFile, testRel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := setup(t, tc.manifest, tc.html)
			for name, call := range map[string]func() error{
				"Load":   func() error { _, err := Load(meta, tc.model); return err },
				"Handle": func() error { _, handled, err := Handle(cmd, meta, tc.model); _ = handled; return err },
			} {
				err := call()
				if !errors.Is(err, tc.want) || !strings.Contains(err.Error(), tc.mention) {
					t.Errorf("%s: got %v, want %v mentioning %q", name, err, tc.want, tc.mention)
				}
			}
		})
	}
	t.Run("missing manifest", func(t *testing.T) {
		ClearCache()
		meta := filepath.Join(t.TempDir(), "meta.json")
		_, err := Load(meta, testModel)
		if !errors.Is(err, ErrMissingFile) || !strings.Contains(err.Error(), meta) {
			t.Errorf("got %v", err)
		}
	})
}
