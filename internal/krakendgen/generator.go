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

	endpoints := make([]Endpoint, 0, len(httpMethods))
	for _, hm := range httpMethods {
		epConfig := getEndpointConfig(hm.method)

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

		ep.InputHeaders = deriveInputHeaders(service, hm.method)
		ep.InputQueryStrings = deriveInputQueryStrings(hm.method)
		ep.ExtraConfig = buildEndpointExtraConfig(gwConfig, epConfig)
		ep.Backend[0].ExtraConfig = buildBackendExtraConfig(gwConfig, epConfig)

		endpoints = append(endpoints, ep)
	}

	serviceName := string(service.Desc.Name())
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
	if rl.GetStrategy() != "" {
		m["strategy"] = rl.GetStrategy()
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

// buildEndpointExtraConfig builds the endpoint-level extra_config map.
// Returns nil if no extra config is needed (so omitempty omits it).
func buildEndpointExtraConfig(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) map[string]any {
	m := make(map[string]any)

	if rl := resolveRateLimit(gwConfig, epConfig); rl != nil {
		m[NamespaceRateLimitRouter] = buildRateLimitRouterConfig(rl)
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

	if len(m) == 0 {
		return nil
	}
	return m
}
