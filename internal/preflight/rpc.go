package preflight

import (
	"context"
	"fmt"

	"github.com/imbanytuidoter/base-node-helper/internal/rpc"
)

type RPCCheck struct {
	URL             string
	ExpectedChainID uint64
}

func (r *RPCCheck) Name() string { return "L1 RPC health" }

func (r *RPCCheck) Run(ctx context.Context) (Result, error) {
	cl, err := rpc.NewL1(r.URL)
	if err != nil {
		return Result{Status: Fail, Message: err.Error()}, nil
	}
	id, err := cl.ChainID(ctx)
	if err != nil {
		return Result{Status: Fail, Message: fmt.Sprintf("chainId failed: %v", err), Fix: "check the URL and provider"}, nil
	}
	if r.ExpectedChainID != 0 && id != r.ExpectedChainID {
		return Result{Status: Fail, Message: fmt.Sprintf("chainId=%d, expected %d", id, r.ExpectedChainID)}, nil
	}
	syncing, err := cl.Syncing(ctx)
	if err != nil {
		return Result{Status: Warn, Message: fmt.Sprintf("eth_syncing failed: %v", err)}, nil
	}
	if syncing {
		return Result{Status: Warn, Message: fmt.Sprintf("L1 chainId=%d but provider is syncing — your Base node will lag", id)}, nil
	}
	return Result{Status: Pass, Message: fmt.Sprintf("L1 chainId=%d, in-sync", id)}, nil
}
