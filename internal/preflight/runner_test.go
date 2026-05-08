package preflight

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubCheck struct {
	name   string
	result Result
	err    error
}

func (s stubCheck) Name() string { return s.name }
func (s stubCheck) Run(ctx context.Context) (Result, error) {
	return s.result, s.err
}

func TestRunnerAggregatesAndReportsWorst(t *testing.T) {
	checks := []Check{
		stubCheck{name: "a", result: Result{Status: Pass, Message: "ok"}},
		stubCheck{name: "b", result: Result{Status: Warn, Message: "slow disk"}},
		stubCheck{name: "c", result: Result{Status: Fail, Message: "no docker"}},
	}
	report := Run(context.Background(), checks)
	if report.Worst() != Fail {
		t.Errorf("worst=%v", report.Worst())
	}
	if len(report.Results) != 3 {
		t.Errorf("len=%d", len(report.Results))
	}
}

func TestRunnerCapturesError(t *testing.T) {
	checks := []Check{
		stubCheck{name: "x", err: errors.New("boom")},
	}
	report := Run(context.Background(), checks)
	if report.Results[0].Status != Fail {
		t.Errorf("status=%v", report.Results[0].Status)
	}
	if !strings.Contains(report.Results[0].Message, "boom") {
		t.Errorf("message=%q", report.Results[0].Message)
	}
}
