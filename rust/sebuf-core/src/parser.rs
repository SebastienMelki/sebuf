use prost_types::{
    DescriptorProto, FieldDescriptorProto, FileDescriptorProto,
    ServiceDescriptorProto,
};
use heck::{ToUpperCamelCase, ToSnakeCase};

#[cfg(test)]
mod tests;

pub struct ProtoParser {
    files: Vec<FileDescriptorProto>,
}

impl ProtoParser {
    pub fn new(files: Vec<FileDescriptorProto>) -> Self {
        Self { files }
    }
    
    pub fn files(&self) -> &[FileDescriptorProto] {
        &self.files
    }
    
    pub fn find_message(&self, name: &str) -> Option<&DescriptorProto> {
        for file in &self.files {
            for message in &file.message_type {
                if message.name.as_deref() == Some(name) {
                    return Some(message);
                }
            }
        }
        None
    }
    
    pub fn find_service(&self, name: &str) -> Option<&ServiceDescriptorProto> {
        for file in &self.files {
            for service in &file.service {
                if service.name.as_deref() == Some(name) {
                    return Some(service);
                }
            }
        }
        None
    }
}

pub struct TypeMapper;

impl TypeMapper {
    pub fn field_to_rust_type(field: &FieldDescriptorProto) -> String {
        use prost_types::field_descriptor_proto::{Label, Type};
        
        let base_type = match field.r#type() {
            Type::Double => "f64".to_string(),
            Type::Float => "f32".to_string(),
            Type::Int64 => "i64".to_string(),
            Type::Uint64 => "u64".to_string(),
            Type::Int32 => "i32".to_string(),
            Type::Fixed64 => "u64".to_string(),
            Type::Fixed32 => "u32".to_string(),
            Type::Bool => "bool".to_string(),
            Type::String => "String".to_string(),
            Type::Bytes => "Vec<u8>".to_string(),
            Type::Uint32 => "u32".to_string(),
            Type::Sfixed32 => "i32".to_string(),
            Type::Sfixed64 => "i64".to_string(),
            Type::Sint32 => "i32".to_string(),
            Type::Sint64 => "i64".to_string(),
            Type::Message | Type::Enum | Type::Group => {
                field.type_name.as_deref().unwrap_or("Unknown").to_string()
            }
        };
        
        match field.label() {
            Label::Optional => format!("Option<{}>", base_type),
            Label::Repeated => format!("Vec<{}>", base_type),
            Label::Required => base_type,
        }
    }
    
    pub fn message_name_to_rust(name: &str) -> String {
        name.to_upper_camel_case()
    }
    
    pub fn field_name_to_rust(name: &str) -> String {
        name.to_snake_case()
    }
    
    pub fn method_name_to_rust(name: &str) -> String {
        name.to_snake_case()
    }
}