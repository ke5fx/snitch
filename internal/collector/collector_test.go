package collector

import (
	"testing"
)

func TestGetConnections(t *testing.T) {
	// integration test to verify /proc parsing works
	conns, err := GetConnections()
	if err != nil {
		t.Fatalf("GetConnections() returned an error: %v", err)
	}

	// connections are dynamic, so just verify function succeeded
	t.Logf("Successfully got %d connections", len(conns))
}