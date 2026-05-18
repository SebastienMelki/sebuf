use anyhow::Result;
use heck::ToSnakeCase;
use prost_types::{
    compiler::code_generator_response,
    DescriptorProto, EnumDescriptorProto, FieldDescriptorProto,
    FileDescriptorProto, MethodDescriptorProto, ServiceDescriptorProto,
};
use std::collections::HashMap;

use crate::schema::*;

pub struct OpenApiGenerator {
    file: FileDescriptorProto,
    all_files: Vec<FileDescriptorProto>,
}

impl OpenApiGenerator {
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
                let spec = self.generate_service_spec(service)?;
                
                let yaml_content = serde_yaml::to_string(&spec)?;
                let output_name = format!("{}.openapi.yaml", service_name);
                
                files.push(code_generator_response::File {
                    name: Some(output_name),
                    content: Some(yaml_content),
                    ..Default::default()
                });
            }
        }
        
        Ok(files)
    }
    
    fn generate_service_spec(&self, service: &ServiceDescriptorProto) -> Result<OpenApiSpec> {
        let service_name = service.name.as_deref().unwrap_or("UnknownService");
        
        let mut spec = OpenApiSpec {
            openapi: "3.1.0".to_string(),
            info: Info {
                title: format!("{} API", service_name),
                version: "1.0.0".to_string(),
                description: Some(format!("API specification for {}", service_name)),
            },
            paths: HashMap::new(),
            components: Some(Components {
                schemas: HashMap::new(),
            }),
            servers: Some(vec![Server {
                url: "http://localhost:8080".to_string(),
                description: Some("Local development server".to_string()),
            }]),
        };
        
        for method in &service.method {
            self.add_method_to_spec(&mut spec, service_name, method)?;
        }
        
        self.collect_message_schemas(&mut spec)?;
        
        Ok(spec)
    }
    
    fn add_method_to_spec(
        &self,
        spec: &mut OpenApiSpec,
        service_name: &str,
        method: &MethodDescriptorProto,
    ) -> Result<()> {
        let method_name = method.name.as_deref().unwrap_or("");
        let path = format!("/api/v1/{}", method_name.to_snake_case());
        
        let input_type = self.resolve_type_name(method.input_type.as_deref().unwrap_or(""));
        let output_type = self.resolve_type_name(method.output_type.as_deref().unwrap_or(""));
        
        let operation = Operation {
            summary: Some(method_name.to_string()),
            description: None,
            operation_id: Some(format!("{}_{}", service_name, method_name)),
            tags: Some(vec![service_name.to_string()]),
            parameters: None,
            request_body: Some(RequestBody {
                required: true,
                content: {
                    let mut content = HashMap::new();
                    content.insert(
                        "application/json".to_string(),
                        MediaType {
                            schema: Schema {
                                reference: Some(format!("#/components/schemas/{}", input_type)),
                                ..Default::default()
                            },
                        },
                    );
                    content
                },
                description: None,
            }),
            responses: {
                let mut responses = HashMap::new();
                responses.insert(
                    "200".to_string(),
                    Response {
                        description: "Successful response".to_string(),
                        content: Some({
                            let mut content = HashMap::new();
                            content.insert(
                                "application/json".to_string(),
                                MediaType {
                                    schema: Schema {
                                        reference: Some(format!("#/components/schemas/{}", output_type)),
                                        ..Default::default()
                                    },
                                },
                            );
                            content
                        }),
                    },
                );
                responses.insert(
                    "400".to_string(),
                    Response {
                        description: "Bad request".to_string(),
                        content: None,
                    },
                );
                responses.insert(
                    "500".to_string(),
                    Response {
                        description: "Internal server error".to_string(),
                        content: None,
                    },
                );
                responses
            },
        };
        
        let path_item = PathItem {
            get: None,
            post: Some(operation),
            put: None,
            delete: None,
            patch: None,
        };
        
        spec.paths.insert(path, path_item);
        Ok(())
    }
    
    fn collect_message_schemas(&self, spec: &mut OpenApiSpec) -> Result<()> {
        let components = spec.components.as_mut().unwrap();
        
        for message in &self.file.message_type {
            if let Some(name) = &message.name {
                let schema = self.message_to_schema(message)?;
                components.schemas.insert(name.clone(), schema);
            }
        }
        
        for enum_type in &self.file.enum_type {
            if let Some(name) = &enum_type.name {
                let schema = self.enum_to_schema(enum_type)?;
                components.schemas.insert(name.clone(), schema);
            }
        }
        
        Ok(())
    }
    
    fn message_to_schema(&self, message: &DescriptorProto) -> Result<Schema> {
        let mut properties = HashMap::new();
        let mut required = Vec::new();
        
        for field in &message.field {
            if let Some(field_name) = &field.name {
                let field_schema = self.field_to_schema(field)?;
                properties.insert(field_name.clone(), field_schema);
                
                if !field.proto3_optional.unwrap_or(false) 
                    && field.label() != prost_types::field_descriptor_proto::Label::Optional {
                    required.push(field_name.clone());
                }
            }
        }
        
        Ok(Schema {
            schema_type: Some("object".to_string()),
            properties: Some(properties),
            required: if required.is_empty() { None } else { Some(required) },
            ..Default::default()
        })
    }
    
    fn enum_to_schema(&self, enum_type: &EnumDescriptorProto) -> Result<Schema> {
        let values: Vec<serde_json::Value> = enum_type
            .value
            .iter()
            .filter_map(|v| v.name.as_ref())
            .map(|name| serde_json::Value::String(name.clone()))
            .collect();
        
        Ok(Schema {
            schema_type: Some("string".to_string()),
            enum_values: Some(values),
            ..Default::default()
        })
    }
    
    fn field_to_schema(&self, field: &FieldDescriptorProto) -> Result<Schema> {
        use prost_types::field_descriptor_proto::{Label, Type};
        
        let base_schema = match field.r#type() {
            Type::Double | Type::Float => Schema {
                schema_type: Some("number".to_string()),
                format: Some(if field.r#type() == Type::Float { "float" } else { "double" }.to_string()),
                ..Default::default()
            },
            Type::Int64 | Type::Uint64 | Type::Sint64 | Type::Fixed64 | Type::Sfixed64 => Schema {
                schema_type: Some("integer".to_string()),
                format: Some("int64".to_string()),
                ..Default::default()
            },
            Type::Int32 | Type::Uint32 | Type::Sint32 | Type::Fixed32 | Type::Sfixed32 => Schema {
                schema_type: Some("integer".to_string()),
                format: Some("int32".to_string()),
                ..Default::default()
            },
            Type::Bool => Schema {
                schema_type: Some("boolean".to_string()),
                ..Default::default()
            },
            Type::String => Schema {
                schema_type: Some("string".to_string()),
                ..Default::default()
            },
            Type::Bytes => Schema {
                schema_type: Some("string".to_string()),
                format: Some("byte".to_string()),
                ..Default::default()
            },
            Type::Message | Type::Enum => {
                let type_name = self.resolve_type_name(field.type_name.as_deref().unwrap_or(""));
                Schema {
                    reference: Some(format!("#/components/schemas/{}", type_name)),
                    ..Default::default()
                }
            },
            Type::Group => Schema {
                schema_type: Some("object".to_string()),
                ..Default::default()
            },
        };
        
        Ok(match field.label() {
            Label::Repeated => Schema {
                schema_type: Some("array".to_string()),
                items: Some(Box::new(base_schema)),
                ..Default::default()
            },
            _ => base_schema,
        })
    }
    
    fn resolve_type_name(&self, type_name: &str) -> String {
        type_name
            .trim_start_matches('.')
            .split('.')
            .last()
            .unwrap_or(type_name)
            .to_string()
    }
}