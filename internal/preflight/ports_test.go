package preflight

import (
	"context"
	"net"
	"testing"
)

func TestPortsCheckDetectsConflict(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	c := &PortsCheck{TCPPorts: []int{port}, UDPPorts: nil}
	r, _ := c.Run(context.Background())
	if r.Status != Fail {
		t.Errorf("status=%v msg=%q", r.Status, r.Message)
	}
}

func TestPortsCheckHappy(t *testing.T) {
	c := &PortsCheck{TCPPorts: []int{0}}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Errorf("status=%v", r.Status)
	}
}
