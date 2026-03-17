package krakendgen

// KrakenDConfig is the top-level wrapper for a standalone KrakenD configuration
// file. Each generated .krakend.json file is a complete, validatable config.
type KrakenDConfig struct {
	Schema    string     `json:"$schema"`
	Version   int        `json:"version"`
	Endpoints []Endpoint `json:"endpoints"`
}

// Endpoint represents a single KrakenD endpoint configuration.
type Endpoint struct {
	Endpoint          string         `json:"endpoint"`
	Method            string         `json:"method"`
	OutputEncoding    string         `json:"output_encoding"`
	Timeout           string         `json:"timeout,omitempty"`
	ConcurrentCalls   int32          `json:"concurrent_calls,omitempty"`
	InputHeaders      []string       `json:"input_headers,omitempty"`
	InputQueryStrings []string       `json:"input_query_strings,omitempty"`
	Backend           []Backend      `json:"backend"`
	ExtraConfig       map[string]any `json:"extra_config,omitempty"`

	// Template-only metadata (excluded from JSON output).
	ServiceName     string `json:"-"` // proto service name for host variable derivation
	HasJWT          bool   `json:"-"` // emit {{ template "jwt_auth_validator.tmpl" . }}
	HasRecaptcha    bool   `json:"-"` // emit {{ include "recaptcha_validator.tmpl" }}
	HeaderPartial   string `json:"-"` // if set, {{ include "xxx" }} instead of inline headers
	IsMethodTimeout bool   `json:"-"` // true when timeout came from endpoint_config
}

// Backend represents a KrakenD backend configuration.
type Backend struct {
	URLPattern  string         `json:"url_pattern"`
	Host        []string       `json:"host"`
	Method      string         `json:"method"`
	Encoding    string         `json:"encoding"`
	ExtraConfig map[string]any `json:"extra_config,omitempty"`
}
