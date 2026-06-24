package net

import "testing"

func TestValidateHostRejectsEmptyAllowlist(t *testing.T) {
	if err := validateHost(nil, "127.0.0.1:5432"); err == nil {
		t.Fatal("expected empty allowedHosts to reject outbound dial")
	}
}

func TestValidateHostAllowsExactHostPort(t *testing.T) {
	if err := validateHost([]string{"api.example.test"}, "api.example.test:443"); err != nil {
		t.Fatalf("expected exact host to be allowed: %v", err)
	}
}

func TestValidateHostAllowsCIDR(t *testing.T) {
	if err := validateHost([]string{"10.0.0.0/24"}, "10.0.0.10:443"); err != nil {
		t.Fatalf("expected CIDR host to be allowed: %v", err)
	}
}
