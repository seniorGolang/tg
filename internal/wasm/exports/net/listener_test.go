package net

import "testing"

func TestValidateListenerRequiresExplicitAllowlist(t *testing.T) {
	if err := validateListener(nil, "tcp", "127.0.0.1:8080"); err == nil {
		t.Fatal("expected empty allowedListeners to reject bind")
	}
}

func TestValidateListenerAllowsExactNetworkAddress(t *testing.T) {
	err := validateListener([]string{"tcp/127.0.0.1:8080"}, "tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("expected listener to be allowed: %v", err)
	}
}

func TestValidateListenerAllowsExplicitGlob(t *testing.T) {
	err := validateListener([]string{"tcp/127.0.0.1:*"}, "tcp", "127.0.0.1:9090")
	if err != nil {
		t.Fatalf("expected listener glob to be allowed: %v", err)
	}
}

func TestValidateListenerRejectsDifferentNetwork(t *testing.T) {
	if err := validateListener([]string{"tcp/127.0.0.1:8080"}, "udp", "127.0.0.1:8080"); err == nil {
		t.Fatal("expected different network to be rejected")
	}
}
