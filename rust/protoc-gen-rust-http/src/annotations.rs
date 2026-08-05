use prost_types::{MethodOptions, ServiceOptions};

pub struct HttpRule {
    pub method: String,
    pub path: String,
    pub body: Option<String>,
    pub response_body: Option<String>,
}

pub struct ServiceHeaders {
    pub required: Vec<HeaderConfig>,
}

pub struct MethodHeaders {
    pub required: Vec<HeaderConfig>,
}

pub struct HeaderConfig {
    pub name: String,
    pub description: Option<String>,
    pub header_type: String,
    pub required: bool,
    pub format: Option<String>,
    pub example: Option<String>,
}

pub fn parse_http_rule(_options: &MethodOptions) -> Option<HttpRule> {
    Some(HttpRule {
        method: "POST".to_string(),
        path: "/api/v1/default".to_string(),
        body: Some("*".to_string()),
        response_body: None,
    })
}

pub fn parse_service_headers(_options: &ServiceOptions) -> Option<ServiceHeaders> {
    None
}

pub fn parse_method_headers(_options: &MethodOptions) -> Option<MethodHeaders> {
    None
}