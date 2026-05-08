//go:build windows

package preflight

import "golang.org/x/sys/windows"

func availableBytes(path string) (uint64, error) {
	var freeAvail, totalBytes, totalFree uint64
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvail, &totalBytes, &totalFree); err != nil {
		return 0, err
	}
	return freeAvail, nil
}
