package krakendgen

// Endpoint represents a single KrakenD endpoint configuration.
type Endpoint struct {
	Endpoint          string    `json:"endpoint"`
	Method            string    `json:"method"`
	OutputEncoding    string    `json:"output_encoding"`
	Timeout           string    `json:"timeout,omitempty"`
	InputHeaders      []string  `json:"input_headers,omitempty"`
	InputQueryStrings []string  `json:"input_query_strings,omitempty"`
	Backend           []Backend `json:"backend"`
}

// Backend represents a KrakenD backend configuration.
type Backend struct {
	URLPattern string   `json:"url_pattern"`
	Host       []string `json:"host"`
	Method     string   `json:"method"`
	Encoding   string   `json:"encoding"`
}
