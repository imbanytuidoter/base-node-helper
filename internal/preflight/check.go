package preflight

import "context"

type Status int

const (
	Pass Status = iota
	Warn
	Fail
)

func (s Status) String() string {
	switch s {
	case Pass:
		return "PASS"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	}
	return "?"
}

type Result struct {
	Name    string
	Status  Status
	Message string
	Fix     string
}

type Check interface {
	Name() string
	Run(ctx context.Context) (Result, error)
}

type Report struct {
	Results []Result
}

func (r Report) Worst() Status {
	worst := Pass
	for _, x := range r.Results {
		if x.Status > worst {
			worst = x.Status
		}
	}
	return worst
}
