package preflight

import (
	"context"
	"fmt"
)

// Run executes checks sequentially. Order matters because some checks depend
// on others (e.g. Compose detection precedes anything that uses compose).
func Run(ctx context.Context, checks []Check) Report {
	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		r, err := c.Run(ctx)
		if r.Name == "" {
			r.Name = c.Name()
		}
		if err != nil {
			r.Status = Fail
			if r.Message == "" {
				r.Message = fmt.Sprintf("(error: %v)", err)
			} else {
				r.Message = fmt.Sprintf("%s (error: %v)", r.Message, err)
			}
		}
		out = append(out, r)
	}
	return Report{Results: out}
}
