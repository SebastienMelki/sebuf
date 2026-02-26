package krakendgen

const (
	NamespaceRateLimitRouter = "qos/ratelimit/router"
	NamespaceRateLimitProxy  = "qos/ratelimit/proxy"
	NamespaceAuthValidator   = "auth/validator"
	NamespaceCircuitBreaker  = "qos/circuit-breaker"
	NamespaceHTTPCache       = "qos/http-cache"
)

var KnownNamespaces = []string{ //nolint:gochecknoglobals // package-level registry used by tests
	NamespaceRateLimitRouter,
	NamespaceRateLimitProxy,
	NamespaceAuthValidator,
	NamespaceCircuitBreaker,
	NamespaceHTTPCache,
}
