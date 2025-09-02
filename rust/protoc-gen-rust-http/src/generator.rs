use anyhow::Result;
use heck::{ToSnakeCase, ToUpperCamelCase};
use prost_types::{
    compiler::code_generator_response,
    FileDescriptorProto, ServiceDescriptorProto,
};
use quote::{format_ident, quote};
use sebuf_core::CodeGenerator;

use crate::annotations::{parse_http_rule, parse_service_headers};

pub struct HttpGenerator {
    file: FileDescriptorProto,
    all_files: Vec<FileDescriptorProto>,
}

impl HttpGenerator {
    pub fn new(file: FileDescriptorProto, all_files: &[FileDescriptorProto]) -> Self {
        Self {
            file,
            all_files: all_files.to_vec(),
        }
    }
    
    pub fn generate(&self) -> Result<Vec<code_generator_response::File>> {
        let mut files = Vec::new();
        
        for service in &self.file.service {
            if let Some(service_name) = &service.name {
                let mut code_gen = CodeGenerator::new();
                
                code_gen.add_import(quote! {
                    use axum::{
                        extract::{Path, Query, State},
                        http::StatusCode,
                        response::IntoResponse,
                        routing::{get, post, put, delete},
                        Json, Router,
                    };
                });
                
                code_gen.add_import(quote! {
                    use serde::{Deserialize, Serialize};
                });
                
                code_gen.add_import(quote! {
                    use std::sync::Arc;
                });
                
                code_gen.add_import(quote! {
                    use tower::ServiceBuilder;
                });
                
                code_gen.add_import(quote! {
                    use tower_http::cors::CorsLayer;
                });
                
                self.generate_service_trait(&mut code_gen, service)?;
                self.generate_router(&mut code_gen, service)?;
                self.generate_handlers(&mut code_gen, service)?;
                
                if let Some(ref options) = service.options {
                    if let Some(service_headers) = parse_service_headers(options) {
                        self.generate_header_middleware(&mut code_gen, &service_headers.required)?;
                    }
                }
                
                let output_name = format!(
                    "{}.http.rs",
                    service_name.to_snake_case()
                );
                
                files.push(code_generator_response::File {
                    name: Some(output_name),
                    content: Some(code_gen.generate()),
                    ..Default::default()
                });
            }
        }
        
        Ok(files)
    }
    
    fn generate_service_trait(
        &self,
        code_gen: &mut CodeGenerator,
        service: &ServiceDescriptorProto,
    ) -> Result<()> {
        let service_name = service.name.as_deref().unwrap_or("UnknownService");
        let trait_name = format_ident!("{}Server", service_name);
        
        let methods: Vec<_> = service.method.iter().map(|method| {
            let method_name = format_ident!("{}", method.name.as_deref().unwrap_or("").to_snake_case());
            let input_type = self.resolve_type_name(method.input_type.as_deref().unwrap_or(""));
            let output_type = self.resolve_type_name(method.output_type.as_deref().unwrap_or(""));
            
            quote! {
                async fn #method_name(&self, request: #input_type) -> Result<#output_type, StatusCode>;
            }
        }).collect();
        
        let trait_def = quote! {
            #[async_trait::async_trait]
            pub trait #trait_name: Send + Sync + 'static {
                #(#methods)*
            }
        };
        
        code_gen.add_item(trait_def);
        Ok(())
    }
    
    fn generate_router(
        &self,
        code_gen: &mut CodeGenerator,
        service: &ServiceDescriptorProto,
    ) -> Result<()> {
        let service_name = service.name.as_deref().unwrap_or("UnknownService");
        let trait_name = format_ident!("{}Server", service_name);
        let router_fn = format_ident!("register_{}_server", service_name.to_snake_case());
        
        let routes: Vec<_> = service.method.iter().map(|method| {
            let handler_name = format_ident!("{}_handler", method.name.as_deref().unwrap_or("").to_snake_case());
            let http_rule = method.options.as_ref()
                .and_then(|opts| parse_http_rule(opts))
                .unwrap_or_else(|| {
                crate::annotations::HttpRule {
                    method: "POST".to_string(),
                    path: format!("/api/v1/{}", method.name.as_deref().unwrap_or("").to_snake_case()),
                    body: Some("*".to_string()),
                    response_body: None,
                }
            });
            
            let path = &http_rule.path;
            let method_str = http_rule.method.to_lowercase();
            let method_fn = format_ident!("{}", method_str);
            
            quote! {
                .route(#path, #method_fn(#handler_name::<S>))
            }
        }).collect();
        
        let router_impl = quote! {
            pub fn #router_fn<S: #trait_name>(server: Arc<S>) -> Router {
                Router::new()
                    #(#routes)*
                    .layer(
                        ServiceBuilder::new()
                            .layer(CorsLayer::permissive())
                            .into_inner()
                    )
                    .with_state(server)
            }
        };
        
        code_gen.add_item(router_impl);
        Ok(())
    }
    
    fn generate_handlers(
        &self,
        code_gen: &mut CodeGenerator,
        service: &ServiceDescriptorProto,
    ) -> Result<()> {
        let service_name = service.name.as_deref().unwrap_or("UnknownService");
        let trait_name = format_ident!("{}Server", service_name);
        
        for method in &service.method {
            let method_name = method.name.as_deref().unwrap_or("");
            let handler_name = format_ident!("{}_handler", method_name.to_snake_case());
            let trait_method = format_ident!("{}", method_name.to_snake_case());
            
            let input_type = self.resolve_type_name(method.input_type.as_deref().unwrap_or(""));
            let _output_type = self.resolve_type_name(method.output_type.as_deref().unwrap_or(""));
            
            let handler = quote! {
                async fn #handler_name<S: #trait_name>(
                    State(server): State<Arc<S>>,
                    Json(request): Json<#input_type>,
                ) -> impl IntoResponse {
                    match server.#trait_method(request).await {
                        Ok(response) => (StatusCode::OK, Json(response)).into_response(),
                        Err(status) => (status, Json(serde_json::json!({
                            "error": status.to_string()
                        }))).into_response(),
                    }
                }
            };
            
            code_gen.add_item(handler);
        }
        
        Ok(())
    }
    
    fn generate_header_middleware(
        &self,
        code_gen: &mut CodeGenerator,
        headers: &[crate::annotations::HeaderConfig],
    ) -> Result<()> {
        let validations: Vec<_> = headers.iter().map(|header| {
            let name = &header.name;
            let required = header.required;
            
            if required {
                quote! {
                    if !headers.contains_key(#name) {
                        return Err((StatusCode::BAD_REQUEST, format!("Missing required header: {}", #name)));
                    }
                }
            } else {
                quote! {}
            }
        }).collect();
        
        let middleware = quote! {
            pub async fn validate_headers(
                headers: axum::http::HeaderMap,
                request: axum::http::Request<axum::body::Body>,
                next: axum::middleware::Next,
            ) -> Result<impl IntoResponse, (StatusCode, String)> {
                #(#validations)*
                Ok(next.run(request).await)
            }
        };
        
        code_gen.add_item(middleware);
        Ok(())
    }
    
    fn resolve_type_name(&self, type_name: &str) -> proc_macro2::TokenStream {
        let clean_name = type_name
            .trim_start_matches('.')
            .split('.')
            .last()
            .unwrap_or(type_name)
            .to_upper_camel_case();
        
        let ident = format_ident!("{}", clean_name);
        quote! { #ident }
    }
}