package cli

import (
	"fmt"
	"net"
	"testing"
)

func TestFindAvailablePort_Default(t *testing.T) {
	port, err := findAvailablePort(0)
	if err != nil {
		t.Fatalf("findAvailablePort(0) failed: %v", err)
	}
	if port < 2021 || port > 2220 {
		t.Errorf("expected port in range [2021, 2220], got %d", port)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Errorf("port %d should be available but got: %v", port, err)
	} else {
		listener.Close()
	}
}

func TestFindAvailablePort_Specific(t *testing.T) {
	port, err := findAvailablePort(12000)
	if err != nil {
		t.Fatalf("findAvailablePort(12000) failed: %v", err)
	}
	if port < 12000 || port > 12099 {
		t.Errorf("expected port in range [12000, 12099], got %d", port)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Errorf("port %d should be available but got: %v", port, err)
	} else {
		listener.Close()
	}
}

func TestFindAvailablePort_TakesNextAvailable(t *testing.T) {
	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		t.Skipf("cannot bind to port 12345 for test: %v", err)
	}
	defer listener.Close()

	port, err := findAvailablePort(12345)
	if err != nil {
		t.Fatalf("findAvailablePort(12345) failed: %v", err)
	}
	if port == 12345 {
		t.Error("expected a different port since 12345 is occupied")
	}
}
