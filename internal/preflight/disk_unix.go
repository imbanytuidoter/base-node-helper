//go:build !windows

package preflight

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func availableBytes(path string) (uint64, error) {
	p := path
	for {
		var s unix.Statfs_t
		if err := unix.Statfs(p, &s); err == nil {
			return s.Bavail * uint64(s.Bsize), nil
		}
		parent := filepath.Dir(p)
		if parent == p {
			return 0, fmt.Errorf("no mountpoint found for %s", path)
		}
		p = parent
	}
}
