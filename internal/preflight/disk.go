package preflight

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"time"
)

type DiskSpaceCheck struct {
	Path          string
	RequiredBytes int64
}

func (d *DiskSpaceCheck) Name() string { return "disk space" }

func (d *DiskSpaceCheck) Run(ctx context.Context) (Result, error) {
	avail, err := availableBytes(d.Path)
	if err != nil {
		return Result{Status: Warn, Message: fmt.Sprintf("could not stat %s: %v", d.Path, err)}, nil
	}
	if avail < uint64(d.RequiredBytes) {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("only %d GiB free at %s, need %d GiB", avail>>30, d.Path, d.RequiredBytes>>30),
			Fix:     "free up space or pick a different data_dir",
		}, nil
	}
	return Result{Status: Pass, Message: fmt.Sprintf("%d GiB free at %s", avail>>30, d.Path)}, nil
}

type DiskSpeedCheck struct {
	Path        string
	SampleBytes int64
	P99WarnNs   int64
	P99FailNs   int64
}

func (d *DiskSpeedCheck) Name() string { return "disk speed (advisory)" }

// maxSampleBytes caps the disk benchmark file size to prevent OOM/DoS from
// a crafted profile with an unreasonably large sample_bytes value.
const maxSampleBytes int64 = 1 << 30 // 1 GiB

func (d *DiskSpeedCheck) Run(ctx context.Context) (Result, error) {
	sampleBytes := d.SampleBytes
	if sampleBytes == 0 {
		sampleBytes = 256 << 20
	}
	// [MED] oom: cap sample size so a crafted profile cannot exhaust disk/memory.
	if sampleBytes > maxSampleBytes {
		sampleBytes = maxSampleBytes
	}
	p99WarnNs := d.P99WarnNs
	if p99WarnNs == 0 {
		p99WarnNs = 200_000
	}
	p99FailNs := d.P99FailNs
	if p99FailNs == 0 {
		p99FailNs = 1_000_000
	}
	if sampleBytes <= 4096 {
		return Result{Status: Warn, Message: "sample size too small for random-read benchmark"}, nil
	}

	f, err := os.CreateTemp(d.Path, ".bnh-disk-bench-*")
	if err != nil {
		return Result{Status: Warn, Message: fmt.Sprintf("cannot write to %s: %v", d.Path, err)}, nil
	}
	defer os.Remove(f.Name())
	buf := make([]byte, 4096)
	if _, err := rand.Read(buf); err != nil {
		f.Close()
		return Result{Status: Warn, Message: err.Error()}, nil
	}
	for written := int64(0); written < sampleBytes; written += int64(len(buf)) {
		// [LOW] Finding 9: honour context cancellation so Ctrl-C stops the
		// write loop promptly instead of blocking until the 1 GiB file is done.
		select {
		case <-ctx.Done():
			f.Close()
			return Result{Status: Warn, Message: "disk benchmark cancelled by context"}, nil
		default:
		}
		if _, err := f.Write(buf); err != nil {
			f.Close()
			return Result{Status: Warn, Message: err.Error()}, nil
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		// defer os.Remove already handles cleanup; no need for explicit call here.
		return Result{Status: Warn, Message: fmt.Sprintf("fsync failed: %v", err)}, nil
	}

	// [MED] toctou: rewind the already-open file descriptor instead of
	// re-opening by name, eliminating the race window where an adversary
	// could swap the file between Close and Open.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return Result{Status: Warn, Message: fmt.Sprintf("seek failed: %v", err)}, nil
	}

	const samples = 256
	maxOff := big.NewInt(sampleBytes - 4096)
	latencies := make([]int64, 0, samples)
	for i := 0; i < samples; i++ {
		n, _ := rand.Int(rand.Reader, maxOff)
		off := n.Int64()
		t0 := time.Now()
		_, err := f.ReadAt(buf, off)
		dt := time.Since(t0).Nanoseconds()
		if err != nil {
			continue
		}
		if dt < 1 {
			dt = 1 // clamp sub-timer-resolution reads to 1 ns
		}
		latencies = append(latencies, dt)
	}
	f.Close()
	if len(latencies) == 0 {
		return Result{Status: Warn, Message: "no successful reads during benchmark"}, nil
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p99 := latencies[(len(latencies)*99)/100]

	if p99 >= p99FailNs {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("4K random-read p99=%dµs at %s — too slow for Base node", p99/1000, d.Path),
			Fix:     "use NVMe SSD; SATA SSDs and HDDs cause sync stalls and reorgs",
		}, nil
	}
	if p99 >= p99WarnNs {
		return Result{
			Status:  Warn,
			Message: fmt.Sprintf("4K random-read p99=%dµs at %s — borderline", p99/1000, d.Path),
		}, nil
	}
	return Result{Status: Pass, Message: fmt.Sprintf("4K random-read p99=%dµs at %s", p99/1000, d.Path)}, nil
}
