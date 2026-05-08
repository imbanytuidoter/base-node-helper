package compose

import (
	"context"
	"strings"
	"testing"
)

func TestPSParsesJSONOutput(t *testing.T) {
	jsonLine := `{"Service":"op-node","State":"running","Status":"Up 2 hours","ExitCode":0}`
	rr := &recordRunner{out: []byte(jsonLine + "\n")}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	containers, err := c.PS(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("PS: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("got %d containers, want 1", len(containers))
	}
	if containers[0].Service != "op-node" {
		t.Errorf("Service=%q", containers[0].Service)
	}
	if containers[0].State != "running" {
		t.Errorf("State=%q", containers[0].State)
	}
}

func TestPSParsesExitedContainer(t *testing.T) {
	jsonLine := `{"Service":"reth","State":"exited","Status":"Exited (137)","ExitCode":137}`
	rr := &recordRunner{out: []byte(jsonLine + "\n")}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	containers, err := c.PS(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("PS: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("got %d containers", len(containers))
	}
	if containers[0].ExitCode == nil {
		t.Fatal("ExitCode should not be nil for exited container")
	}
	if *containers[0].ExitCode != 137 {
		t.Errorf("ExitCode=%d, want 137", *containers[0].ExitCode)
	}
}

func TestPSEmptyOutput(t *testing.T) {
	rr := &recordRunner{out: []byte("")}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	containers, err := c.PS(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("PS: %v", err)
	}
	if len(containers) != 0 {
		t.Errorf("expected 0 containers, got %d", len(containers))
	}
}

func TestDownWithVolumes(t *testing.T) {
	rr := &recordRunner{}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	err := c.Down(context.Background(), DownOpts{ProjectDir: "/repo", RemoveVolumes: true})
	if err != nil {
		t.Fatalf("Down: %v", err)
	}
	got := strings.Join(rr.calls[0], " ")
	if !strings.Contains(got, "-v") {
		t.Errorf("expected -v flag for remove-volumes, got: %s", got)
	}
}

func TestNewReturnsCompose(t *testing.T) {
	inv := Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}
	c := New(inv)
	if c == nil {
		t.Error("New returned nil")
	}
}
