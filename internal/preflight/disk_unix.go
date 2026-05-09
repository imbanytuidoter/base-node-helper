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
			// [MED] integer-overflow: Bsize can be int32 (macOS) or int64 (Linux).
			// A zero or negative Bsize would produce a nonsensical result.
			if s.Bsize <= 0 {
				return 0, fmt.Errorf("unexpected Bsize %d for path %s", s.Bsize, p)
			}
			avail := s.Bavail * uint64(s.Bsize)
			// Overflow check: if multiplication wrapped, result is smaller than multiplicand.
			if s.Bavail != 0 && avail/s.Bavail != uint64(s.Bsize) {
				return 0, fmt.Errorf("overflow computing available bytes for %s", p)
			}
			return avail, nil
		}
		parent := filepath.Dir(p)
		if parent == p {
			return 0, fmt.Errorf("no mountpoint found for %s", path)
		}
		p = parent
	}
}
