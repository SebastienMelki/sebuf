package krakendgen

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/internal/annotations"
	"github.com/SebastienMelki/sebuf/krakend"
)

// GenerateService reads sebuf.http and sebuf.krakend annotations from a
// protogen service definition and produces a slice of KrakenD Endpoint
// objects. Each RPC with a sebuf.http.config annotation becomes one endpoint;
// RPCs without that annotation are silently skipped.
//
// If at least one RPC has HTTP config, the service must have a
// sebuf.krakend.gateway_config annotation providing a backend host.
// If no RPCs have HTTP config, an empty slice is returned without error.
func GenerateService(service *protogen.Service) ([]Endpoint, error) {
	// Collect RPCs that have HTTP annotations before requiring gateway_config.
	// This avoids failing on services that have nothing to do with HTTP/KrakenD.
	type httpMethod struct {
		method     *protogen.Method
		httpConfig *annotations.HTTPConfig
	}

	var httpMethods []httpMethod
	for _, method := range service.Methods {
		cfg := annotations.GetMethodHTTPConfig(method)
		if cfg != nil {
			httpMethods = append(httpMethods, httpMethod{method: method, httpConfig: cfg})
		}
	}

	// No HTTP-annotated RPCs -- nothing to generate.
	if len(httpMethods) == 0 {
		return nil, nil
	}

	// At least one RPC needs a backend, so gateway_config is required.
	gwConfig, err := getGatewayConfig(service)
	if err != nil {
		return nil, err
	}

	basePath := annotations.GetServiceBasePath(service)
	serviceName := string(service.Desc.Name())

	// Validate service-level rate limit, circuit breaker, and cache configs before building endpoints.
	if rl := gwConfig.GetRateLimit(); rl != nil {
		if err := validateRateLimit(rl, serviceName); err != nil {
			return nil, err
		}
	}
	if cb := gwConfig.GetCircuitBreaker(); cb != nil {
		if err := validateCircuitBreaker(cb, serviceName); err != nil {
			return nil, err
		}
	}
	if cache := gwConfig.GetCache(); cache != nil {
		if err := validateCache(cache, serviceName); err != nil {
			return nil, err
		}
	}

	endpoints := make([]Endpoint, 0, len(httpMethods))
	for _, hm := range httpMethods {
		epConfig := getEndpointConfig(hm.method)

		// Validate method-level rate limit, circuit breaker, and cache if present.
		if epConfig != nil {
			if rl := epConfig.GetRateLimit(); rl != nil {
				if err := validateRateLimit(rl, serviceName); err != nil {
					return nil, err
				}
			}
			if cb := epConfig.GetCircuitBreaker(); cb != nil {
				if err := validateCircuitBreaker(cb, serviceName); err != nil {
					return nil, err
				}
			}
			if cache := epConfig.GetCache(); cache != nil {
				if err := validateCache(cache, serviceName); err != nil {
					return nil, err
				}
			}
		}

		fullPath := annotations.BuildHTTPPath(basePath, hm.httpConfig.Path)
		host := resolveHost(gwConfig, epConfig)
		timeout := resolveTimeout(gwConfig, epConfig)

		ep := Endpoint{
			Endpoint:       fullPath,
			Method:         hm.httpConfig.Method,
			OutputEncoding: "json",
			Backend: []Backend{
				{
					URLPattern: fullPath,
					Host:       host,
					Method:     hm.httpConfig.Method,
					Encoding:   "json",
				},
			},
		}

		if timeout != "" {
			ep.Timeout = timeout
		}

		if cc := resolveConcurrentCalls(gwConfig, epConfig); cc > 0 {
			ep.ConcurrentCalls = cc
		}

		ep.InputHeaders = deriveInputHeaders(service, hm.method)

		// Auto-add JWT propagated claim headers to input_headers.
		if jwt := gwConfig.GetJwt(); jwt != nil {
			propagatedHeaders := getJWTPropagatedHeaderNames(jwt)
			if len(propagatedHeaders) > 0 {
				headers := ep.InputHeaders
				if headers == nil {
					headers = []string{}
				}
				for _, h := range propagatedHeaders {
					if !containsString(headers, h) {
						headers = append(headers, h)
					}
				}
				sort.Strings(headers)
				ep.InputHeaders = headers
			}
		}

		ep.InputQueryStrings = deriveInputQueryStrings(hm.method)
		ep.ExtraConfig = buildEndpointExtraConfig(gwConfig, epConfig)
		ep.Backend[0].ExtraConfig = buildBackendExtraConfig(gwConfig, epConfig)

		endpoints = append(endpoints, ep)
	}

	if err := ValidateRoutes(endpoints, serviceName); err != nil {
		return nil, err
	}

	return endpoints, nil
}

// getGatewayConfig reads the sebuf.krakend.gateway_config extension from the
// service options. Returns an error if the annotation is missing -- backend
// host is required for every service.
func getGatewayConfig(service *protogen.Service) (*krakend.GatewayConfig, error) {
	options := service.Desc.Options()
	if options == nil {
		return nil, fmt.Errorf(
			"service %s has no (sebuf.krakend.gateway_config) annotation -- backend host is required",
			service.Desc.Name(),
		)
	}

	serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
	if !ok {
		return nil, fmt.Errorf(
			"service %s has no (sebuf.krakend.gateway_config) annotation -- backend host is required",
			service.Desc.Name(),
		)
	}

	ext := proto.GetExtension(serviceOptions, krakend.E_GatewayConfig)
	if ext == nil {
		return nil, fmt.Errorf(
			"service %s has no (sebuf.krakend.gateway_config) annotation -- backend host is required",
			service.Desc.Name(),
		)
	}

	gwConfig, ok := ext.(*krakend.GatewayConfig)
	if !ok || gwConfig == nil {
		return nil, fmt.Errorf(
			"service %s has no (sebuf.krakend.gateway_config) annotation -- backend host is required",
			service.Desc.Name(),
		)
	}

	return gwConfig, nil
}

// getEndpointConfig reads the sebuf.krakend.endpoint_config extension from
// method options. Returns nil if the annotation is absent.
func getEndpointConfig(method *protogen.Method) *krakend.EndpointConfig {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

	ext := proto.GetExtension(methodOptions, krakend.E_EndpointConfig)
	if ext == nil {
		return nil
	}

	epConfig, ok := ext.(*krakend.EndpointConfig)
	if !ok {
		return nil
	}

	return epConfig
}

// resolveHost returns the method-level host override if non-empty, otherwise
// the service-level host from gateway_config.
func resolveHost(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) []string {
	if epConfig != nil && len(epConfig.GetHost()) > 0 {
		return epConfig.GetHost()
	}
	return gwConfig.GetHost()
}

// resolveTimeout returns the method-level timeout override if non-empty,
// otherwise the service-level timeout from gateway_config.
func resolveTimeout(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) string {
	if epConfig != nil && epConfig.GetTimeout() != "" {
		return epConfig.GetTimeout()
	}
	return gwConfig.GetTimeout()
}

// deriveInputHeaders extracts header names from service-level and method-level
// header annotations, merges them (method overrides service for same-name
// headers), and returns a sorted list of header names. Returns nil if no
// headers are annotated so that the omitempty JSON tag omits the field.
func deriveInputHeaders(service *protogen.Service, method *protogen.Method) []string {
	serviceHeaders := annotations.GetServiceHeaders(service)
	methodHeaders := annotations.GetMethodHeaders(method)

	combined := annotations.CombineHeaders(serviceHeaders, methodHeaders)
	if len(combined) == 0 {
		return nil
	}

	names := make([]string, 0, len(combined))
	for _, h := range combined {
		if name := h.GetName(); name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return nil
	}

	sort.Strings(names)
	return names
}

// deriveInputQueryStrings extracts query parameter names from the request
// message's sebuf.http.query annotations. Returns nil if no query params are
// annotated so that the omitempty JSON tag omits the field.
func deriveInputQueryStrings(method *protogen.Method) []string {
	params := annotations.GetQueryParams(method.Input)
	if len(params) == 0 {
		return nil
	}

	names := make([]string, 0, len(params))
	for _, qp := range params {
		names = append(names, qp.ParamName)
	}

	sort.Strings(names)
	return names
}

// ---------------------------------------------------------------------------
// Rate limiting resolvers and builders
// ---------------------------------------------------------------------------

// resolveRateLimit returns the method-level rate limit if present, otherwise
// the service-level rate limit. Returns nil if neither has it.
func resolveRateLimit(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) *krakend.RateLimitConfig {
	if epConfig != nil && epConfig.GetRateLimit() != nil {
		return epConfig.GetRateLimit()
	}
	return gwConfig.GetRateLimit()
}

// resolveBackendRateLimit returns the method-level backend rate limit if
// present, otherwise the service-level backend rate limit. Returns nil if
// neither has it.
func resolveBackendRateLimit(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) *krakend.BackendRateLimitConfig {
	if epConfig != nil && epConfig.GetBackendRateLimit() != nil {
		return epConfig.GetBackendRateLimit()
	}
	return gwConfig.GetBackendRateLimit()
}

// buildRateLimitRouterConfig builds the endpoint-level rate limit config map
// for the qos/ratelimit/router namespace. Only includes non-zero fields.
// validateRateLimit verifies that HEADER and PARAM strategies have a key set.
// IP strategy does not need a key (KrakenD reads the client IP automatically).
func validateRateLimit(rl *krakend.RateLimitConfig, serviceName string) error {
	s := rl.GetStrategy()
	if (s == krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_HEADER ||
		s == krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_PARAM) && rl.GetKey() == "" {
		return fmt.Errorf(
			"service %s: rate_limit strategy %s requires a key (header or param name)",
			serviceName, s,
		)
	}
	return nil
}

func buildRateLimitRouterConfig(rl *krakend.RateLimitConfig) map[string]any {
	m := make(map[string]any)

	if rl.GetMaxRate() != 0 {
		m["max_rate"] = rl.GetMaxRate()
	}
	if rl.GetCapacity() != 0 {
		m["capacity"] = rl.GetCapacity()
	}
	if rl.GetEvery() != "" {
		m["every"] = rl.GetEvery()
	}
	if rl.GetClientMaxRate() != 0 {
		m["client_max_rate"] = rl.GetClientMaxRate()
	}
	if rl.GetClientCapacity() != 0 {
		m["client_capacity"] = rl.GetClientCapacity()
	}
	if s := rateLimitStrategyToString(rl.GetStrategy()); s != "" {
		m["strategy"] = s
	}
	if rl.GetKey() != "" {
		m["key"] = rl.GetKey()
	}

	return m
}

// buildBackendRateLimitConfig builds the backend-level rate limit config map
// for the qos/ratelimit/proxy namespace. Only includes non-zero fields.
func buildBackendRateLimitConfig(brl *krakend.BackendRateLimitConfig) map[string]any {
	m := make(map[string]any)

	if brl.GetMaxRate() != 0 {
		m["max_rate"] = brl.GetMaxRate()
	}
	if brl.GetCapacity() != 0 {
		m["capacity"] = brl.GetCapacity()
	}
	if brl.GetEvery() != "" {
		m["every"] = brl.GetEvery()
	}

	return m
}

// ---------------------------------------------------------------------------
// Enum-to-string mapping functions
// ---------------------------------------------------------------------------

// rateLimitStrategyToString converts a RateLimitStrategy enum to the lowercase
// string value expected by KrakenD JSON config.
func rateLimitStrategyToString(s krakend.RateLimitStrategy) string {
	switch s {
	case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_IP:
		return "ip"
	case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_HEADER:
		return "header"
	case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_PARAM:
		return "param"
	default:
		return ""
	}
}

// jwtAlgorithmToString converts a JWTAlgorithm enum to the string value
// expected by KrakenD JSON config.
func jwtAlgorithmToString(a krakend.JWTAlgorithm) string {
	switch a {
	case krakend.JWTAlgorithm_JWT_ALGORITHM_RS256:
		return "RS256"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_RS384:
		return "RS384"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_RS512:
		return "RS512"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_HS256:
		return "HS256"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_HS384:
		return "HS384"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_HS512:
		return "HS512"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_ES256:
		return "ES256"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_ES384:
		return "ES384"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_ES512:
		return "ES512"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_PS256:
		return "PS256"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_PS384:
		return "PS384"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_PS512:
		return "PS512"
	case krakend.JWTAlgorithm_JWT_ALGORITHM_EDDSA:
		return "EdDSA"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// JWT auth/validator resolvers and builders
// ---------------------------------------------------------------------------

// buildAuthValidatorConfig builds the auth/validator config map from a
// JWTConfig proto message. Only includes non-zero/non-empty fields.
func buildAuthValidatorConfig(jwt *krakend.JWTConfig) map[string]any {
	m := make(map[string]any)

	if a := jwtAlgorithmToString(jwt.GetAlg()); a != "" {
		m["alg"] = a
	}
	if jwt.GetJwkUrl() != "" {
		m["jwk_url"] = jwt.GetJwkUrl()
	}
	if len(jwt.GetAudience()) > 0 {
		m["audience"] = jwt.GetAudience()
	}
	if jwt.GetIssuer() != "" {
		m["issuer"] = jwt.GetIssuer()
	}
	if jwt.GetCache() {
		m["cache"] = true
	}
	if claims := buildPropagateClaims(jwt.GetPropagateClaims()); claims != nil {
		m["propagate_claims"] = claims
	}

	return m
}

// buildPropagateClaims converts proto ClaimToHeader messages to KrakenD's
// expected array-of-arrays format: [["claim_name", "Header-Name"], ...].
// Returns nil if input is empty.
func buildPropagateClaims(claims []*krakend.ClaimToHeader) [][]string {
	if len(claims) == 0 {
		return nil
	}

	result := make([][]string, 0, len(claims))
	for _, c := range claims {
		result = append(result, []string{c.GetClaim(), c.GetHeader()})
	}
	return result
}

// getJWTPropagatedHeaderNames extracts just the header names from
// propagate_claims. Used to auto-add them to input_headers.
func getJWTPropagatedHeaderNames(jwt *krakend.JWTConfig) []string {
	claims := jwt.GetPropagateClaims()
	if len(claims) == 0 {
		return nil
	}

	names := make([]string, 0, len(claims))
	for _, c := range claims {
		if h := c.GetHeader(); h != "" {
			names = append(names, h)
		}
	}
	return names
}

// containsString returns true if slice contains s.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Circuit breaker resolvers and builders
// ---------------------------------------------------------------------------

// resolveCircuitBreaker returns the method-level circuit breaker if present,
// otherwise the service-level circuit breaker. Returns nil if neither has it.
func resolveCircuitBreaker(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) *krakend.CircuitBreakerConfig {
	if epConfig != nil && epConfig.GetCircuitBreaker() != nil {
		return epConfig.GetCircuitBreaker()
	}
	return gwConfig.GetCircuitBreaker()
}

// validateCircuitBreaker verifies that all required fields (interval, timeout,
// max_errors) are positive when a circuit breaker config is present.
func validateCircuitBreaker(cb *krakend.CircuitBreakerConfig, serviceName string) error {
	if cb.GetInterval() <= 0 || cb.GetTimeout() <= 0 || cb.GetMaxErrors() <= 0 {
		return fmt.Errorf(
			"service %s: circuit_breaker requires interval, timeout, and max_errors (all must be > 0)",
			serviceName,
		)
	}
	return nil
}

// buildCircuitBreakerConfig builds the backend-level circuit breaker config
// map for the qos/circuit-breaker namespace.
func buildCircuitBreakerConfig(cb *krakend.CircuitBreakerConfig) map[string]any {
	m := make(map[string]any)

	m["interval"] = cb.GetInterval()
	m["timeout"] = cb.GetTimeout()
	m["max_errors"] = cb.GetMaxErrors()

	if cb.GetName() != "" {
		m["name"] = cb.GetName()
	}
	if cb.GetLogStatusChange() {
		m["log_status_change"] = true
	}

	return m
}

// ---------------------------------------------------------------------------
// Backend caching resolvers and builders
// ---------------------------------------------------------------------------

// resolveCache returns the method-level cache if present, otherwise the
// service-level cache. Returns nil if neither has it.
func resolveCache(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) *krakend.CacheConfig {
	if epConfig != nil && epConfig.GetCache() != nil {
		return epConfig.GetCache()
	}
	return gwConfig.GetCache()
}

// validateCache verifies that max_items and max_size are both set or both
// unset. Having only one is a configuration error.
func validateCache(cache *krakend.CacheConfig, serviceName string) error {
	if cache.GetShared() && (cache.GetMaxItems() > 0 || cache.GetMaxSize() > 0) {
		return fmt.Errorf(
			"service %s: cache cannot combine shared with max_items/max_size (KrakenD oneOf constraint)",
			serviceName,
		)
	}
	hasMaxItems := cache.GetMaxItems() > 0
	hasMaxSize := cache.GetMaxSize() > 0
	if hasMaxItems != hasMaxSize {
		return fmt.Errorf(
			"service %s: cache max_items and max_size must both be set or both unset",
			serviceName,
		)
	}
	return nil
}

// buildHTTPCacheConfig builds the backend-level HTTP cache config map for
// the qos/http-cache namespace.
func buildHTTPCacheConfig(cache *krakend.CacheConfig) map[string]any {
	m := make(map[string]any)

	if cache.GetShared() {
		m["shared"] = true
	}
	if cache.GetMaxItems() > 0 && cache.GetMaxSize() > 0 {
		m["max_items"] = cache.GetMaxItems()
		m["max_size"] = cache.GetMaxSize()
	}

	return m
}

// ---------------------------------------------------------------------------
// Concurrent calls resolver
// ---------------------------------------------------------------------------

// resolveConcurrentCalls returns the method-level concurrent calls if > 0,
// otherwise the service-level value. Returns 0 if neither sets it (0 means
// omit via omitempty on the Endpoint struct).
func resolveConcurrentCalls(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) int32 {
	if epConfig != nil && epConfig.GetConcurrentCalls() > 0 {
		return epConfig.GetConcurrentCalls()
	}
	return gwConfig.GetConcurrentCalls()
}

// buildEndpointExtraConfig builds the endpoint-level extra_config map.
// Returns nil if no extra config is needed (so omitempty omits it).
func buildEndpointExtraConfig(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) map[string]any {
	m := make(map[string]any)

	if rl := resolveRateLimit(gwConfig, epConfig); rl != nil {
		m[NamespaceRateLimitRouter] = buildRateLimitRouterConfig(rl)
	}

	// JWT is service-level only -- read from gwConfig, never from epConfig.
	if jwt := gwConfig.GetJwt(); jwt != nil {
		m[NamespaceAuthValidator] = buildAuthValidatorConfig(jwt)
	}

	if len(m) == 0 {
		return nil
	}
	return m
}

// buildBackendExtraConfig builds the backend-level extra_config map.
// Returns nil if no extra config is needed (so omitempty omits it).
func buildBackendExtraConfig(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) map[string]any {
	m := make(map[string]any)

	if brl := resolveBackendRateLimit(gwConfig, epConfig); brl != nil {
		m[NamespaceRateLimitProxy] = buildBackendRateLimitConfig(brl)
	}

	if cb := resolveCircuitBreaker(gwConfig, epConfig); cb != nil {
		m[NamespaceCircuitBreaker] = buildCircuitBreakerConfig(cb)
	}

	if cache := resolveCache(gwConfig, epConfig); cache != nil {
		m[NamespaceHTTPCache] = buildHTTPCacheConfig(cache)
	}

	if len(m) == 0 {
		return nil
	}
	return m
}
