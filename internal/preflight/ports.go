package preflight

import (
	"context"
	"fmt"
	"net"
	"strconv"
)

type PortsCheck struct {
	TCPPorts []int
	UDPPorts []int
}

func NewPortsCheck() *PortsCheck {
	return &PortsCheck{TCPPorts: []int{30303, 9222}, UDPPorts: []int{30303}}
}

func (p *PortsCheck) Name() string { return "ports listening" }

func (p *PortsCheck) Run(ctx context.Context) (Result, error) {
	var conflicts []string
	for _, port := range p.TCPPorts {
		inUse := false
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			inUse = true
		} else {
			ln.Close()
		}
		if inUse {
			conflicts = append(conflicts, fmt.Sprintf("tcp/%d", port))
		}
	}
	for _, port := range p.UDPPorts {
		conn, err := net.ListenPacket("udp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
		if err != nil {
			conflicts = append(conflicts, fmt.Sprintf("udp/%d (%v)", port, err))
			continue
		}
		conn.Close()
	}
	if len(conflicts) > 0 {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("ports in use: %v", conflicts),
			Fix:     "stop the conflicting process (lsof -i :30303 to find it) or change ports in profile",
		}, nil
	}
	return Result{Status: Pass, Message: "P2P/RPC ports free for bind"}, nil
}
