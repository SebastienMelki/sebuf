package krakendgen

import "testing"

func TestNamespaceConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"RateLimitRouter", NamespaceRateLimitRouter, "qos/ratelimit/router"},
		{"RateLimitProxy", NamespaceRateLimitProxy, "qos/ratelimit/proxy"},
		{"AuthValidator", NamespaceAuthValidator, "auth/validator"},
		{"CircuitBreaker", NamespaceCircuitBreaker, "qos/circuit-breaker"},
		{"HTTPCache", NamespaceHTTPCache, "qos/http-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("got %q, want %q", tt.constant, tt.want)
			}
		})
	}
}

func TestKnownNamespacesContainsAll(t *testing.T) {
	expected := []string{
		NamespaceRateLimitRouter,
		NamespaceRateLimitProxy,
		NamespaceAuthValidator,
		NamespaceCircuitBreaker,
		NamespaceHTTPCache,
	}

	if len(KnownNamespaces) != len(expected) {
		t.Fatalf("KnownNamespaces has %d entries, want %d", len(KnownNamespaces), len(expected))
	}

	nsSet := make(map[string]bool, len(KnownNamespaces))
	for _, ns := range KnownNamespaces {
		nsSet[ns] = true
	}

	for _, exp := range expected {
		if !nsSet[exp] {
			t.Errorf("KnownNamespaces missing %q", exp)
		}
	}
}

func TestKnownNamespacesNoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(KnownNamespaces))
	for _, ns := range KnownNamespaces {
		if seen[ns] {
			t.Errorf("duplicate namespace in KnownNamespaces: %q", ns)
		}
		seen[ns] = true
	}
}
