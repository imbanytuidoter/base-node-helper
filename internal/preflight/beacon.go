package preflight

import (
	"context"
	"fmt"

	"github.com/imbanytuidoter/base-node-helper/internal/rpc"
)

type BeaconCheck struct {
	URL string
}

func (b *BeaconCheck) Name() string { return "L1 Beacon health" }

func (b *BeaconCheck) Run(ctx context.Context) (Result, error) {
	cl, err := rpc.NewBeacon(b.URL)
	if err != nil {
		return Result{Status: Fail, Message: err.Error()}, nil
	}
	gt, err := cl.Genesis(ctx)
	if err != nil {
		return Result{Status: Fail, Message: fmt.Sprintf("beacon genesis failed: %v", err), Fix: "use a Beacon endpoint with /eth/v1/beacon/genesis"}, nil
	}
	return Result{Status: Pass, Message: fmt.Sprintf("Beacon reachable, genesis_time=%s", gt)}, nil
}
