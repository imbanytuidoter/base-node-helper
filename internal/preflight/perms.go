package preflight

import (
	"context"
	"fmt"
	"os"
)

type PermsCheck struct {
	Path string
}

func (p *PermsCheck) Name() string { return "data dir permissions" }

func (p *PermsCheck) Run(ctx context.Context) (Result, error) {
	st, err := os.Stat(p.Path)
	if os.IsNotExist(err) {
		return Result{Status: Warn, Message: fmt.Sprintf("%s does not exist (will be created on first start)", p.Path)}, nil
	}
	if err != nil {
		return Result{Status: Fail, Message: err.Error()}, nil
	}
	if !st.IsDir() {
		return Result{Status: Fail, Message: fmt.Sprintf("%s is not a directory", p.Path)}, nil
	}
	f, err := os.CreateTemp(p.Path, ".bnh-write-probe-*")
	if err != nil {
		return Result{Status: Fail, Message: fmt.Sprintf("write probe failed: %v", err), Fix: "chown -R $USER " + p.Path}, nil
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("ok")); err != nil {
		f.Close()
		return Result{Status: Fail, Message: fmt.Sprintf("write probe failed: %v", err), Fix: "chown -R $USER " + p.Path}, nil
	}
	f.Close()
	return Result{Status: Pass, Message: fmt.Sprintf("%s writable", p.Path)}, nil
}
