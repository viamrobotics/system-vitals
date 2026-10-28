// Package customcontrol serves a model's custom control artifact, declared as
// custom_control in the module's meta.json, through the resource's DoCommand.
// The command is transitional: keep meta.json and delete this package once the
// platform serves the declared artifact itself.
package customcontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Command is the DoCommand "command" value this package answers.
const Command = "get-custom-control"

const (
	commandKey  = "command"
	htmlKey     = "html"
	revisionKey = "revision"
	pathKey     = "path"
)

var (
	// ErrNoEntry means meta.json has no models entry for the requested model.
	ErrNoEntry = errors.New("no models entry in meta.json")
	// ErrNoControl means the model entry declares no custom_control path.
	ErrNoControl = errors.New("model entry has no custom_control")
	// ErrMissingFile means the manifest or the declared artifact could not be read.
	ErrMissingFile = errors.New("file missing or unreadable")
)

// Artifact is the DoCommand response payload.
type Artifact struct {
	// HTML is the contents of the artifact file.
	HTML string `json:"html"`
	// Revision is the hex sha256 of the artifact file bytes; the app remounts
	// the card when it changes.
	Revision string `json:"revision"`
	// Path is the custom_control value as written in meta.json.
	Path string `json:"path"`
}

type manifest struct {
	Models []struct {
		Model         string `json:"model"`
		CustomControl string `json:"custom_control"`
	} `json:"models"`
}

type cacheKey struct{ metaPath, model string }

var (
	cacheMu sync.Mutex
	cache   = map[cacheKey]Artifact{}
)

// ClearCache drops every cached artifact so the next Load re-reads from disk.
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = map[cacheKey]Artifact{}
}

// Load resolves the model's custom_control path relative to the directory of
// metaPath, reads it once, and caches the result per (metaPath, model).
func Load(metaPath, model string) (Artifact, error) {
	key := cacheKey{metaPath, model}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if a, ok := cache[key]; ok {
		return a, nil
	}

	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("read manifest %q: %w: %w", metaPath, ErrMissingFile, err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Artifact{}, fmt.Errorf("parse manifest %q: %w", metaPath, err)
	}

	control, found := "", false
	for _, entry := range m.Models {
		if entry.Model == model {
			control, found = entry.CustomControl, true
			break
		}
	}
	if !found {
		return Artifact{}, fmt.Errorf("model %q in %q: %w", model, metaPath, ErrNoEntry)
	}
	if control == "" {
		return Artifact{}, fmt.Errorf("model %q in %q: %w", model, metaPath, ErrNoControl)
	}

	artifactPath := filepath.Join(filepath.Dir(metaPath), filepath.FromSlash(control))
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("model %q artifact %q: %w: %w", model, artifactPath, ErrMissingFile, err)
	}
	sum := sha256.Sum256(data)
	a := Artifact{HTML: string(data), Revision: hex.EncodeToString(sum[:]), Path: control}
	cache[key] = a
	return a, nil
}

// Handle answers the get-custom-control command. handled is false when cmd is
// some other command, so callers can fall through to their own handling.
func Handle(cmd map[string]any, metaPath, model string) (map[string]any, bool, error) {
	if name, _ := cmd[commandKey].(string); name != Command {
		return nil, false, nil
	}
	a, err := Load(metaPath, model)
	if err != nil {
		return nil, true, err
	}
	return map[string]any{htmlKey: a.HTML, revisionKey: a.Revision, pathKey: a.Path}, true, nil
}
