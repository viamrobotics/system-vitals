package systemvitals

import (
	"os"
	"path/filepath"
	"sync"
)

const metaFileName = "meta.json"

// metaPath locates the module's meta.json, which declares the custom control artifact.
// Order: $VIAM_MODULE_ROOT (set by viam-server for local modules and by app.viam.com for
// registry modules), then the directories above the executable (the tarball ships
// bin/<binary> beside meta.json), then the working directory.
var metaPath = sync.OnceValue(func() string {
	if root := os.Getenv("VIAM_MODULE_ROOT"); root != "" {
		if p := filepath.Join(root, metaFileName); fileExists(p) {
			return p
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		const maxLevels = 3
		for i := 0; i < maxLevels; i++ {
			if p := filepath.Join(dir, metaFileName); fileExists(p) {
				return p
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return metaFileName
})

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
