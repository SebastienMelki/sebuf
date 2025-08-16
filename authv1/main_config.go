package authv1

import (
	"net/http"
)

type ServerOption func(c *serverConfiguration)

type serverConfiguration struct {
	mux     *http.ServeMux
	withMux bool
}

func getDefaultConfiguration() *serverConfiguration {
	return &serverConfiguration{
		mux:     http.DefaultServeMux,
		withMux: false,
	}
}

func getConfiguration(options ...ServerOption) *serverConfiguration {
	configuration := getDefaultConfiguration()
	for _, option := range options {
		option(configuration)
	}
	return configuration
}

func WithMux(mux *http.ServeMux) ServerOption {
	return func(c *serverConfiguration) {
		c.mux = mux
		c.withMux = true
	}
}
