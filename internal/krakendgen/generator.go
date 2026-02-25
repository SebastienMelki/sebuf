package krakendgen

import (
	"fmt"

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

		endpoints = append(endpoints, ep)
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
