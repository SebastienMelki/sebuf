use anyhow::Result;
use heck::{ToSnakeCase, ToUpperCamelCase};
use prost_types::{
    compiler::code_generator_response,
    DescriptorProto, FieldDescriptorProto, FileDescriptorProto, OneofDescriptorProto,
};
use quote::{format_ident, quote};
use sebuf_core::CodeGenerator;

pub struct OneofHelperGenerator {
    file: FileDescriptorProto,
}

impl OneofHelperGenerator {
    pub fn new(file: FileDescriptorProto) -> Self {
        Self { file }
    }
    
    pub fn generate(&self) -> Result<Option<code_generator_response::File>> {
        let mut code_gen = CodeGenerator::new();
        let mut has_oneofs = false;
        
        for message in &self.file.message_type {
            if let Some(message_name) = &message.name {
                for (oneof_index, oneof) in message.oneof_decl.iter().enumerate() {
                    if let Some(oneof_name) = &oneof.name {
                        has_oneofs = true;
                        self.generate_oneof_helpers(
                            &mut code_gen,
                            message,
                            message_name,
                            oneof,
                            oneof_name,
                            oneof_index as i32,
                        )?;
                    }
                }
            }
        }
        
        if !has_oneofs {
            return Ok(None);
        }
        
        let package = self.file.package.as_deref().unwrap_or("");
        let _rust_module = package.replace('.', "_");
        let output_name = format!(
            "{}.oneof_helpers.rs",
            self.file.name.as_deref().unwrap_or("unknown").replace(".proto", "")
        );
        
        let generated_code = code_gen.generate();
        
        Ok(Some(code_generator_response::File {
            name: Some(output_name),
            content: Some(generated_code),
            ..Default::default()
        }))
    }
    
    fn generate_oneof_helpers(
        &self,
        code_gen: &mut CodeGenerator,
        message: &DescriptorProto,
        message_name: &str,
        _oneof: &OneofDescriptorProto,
        oneof_name: &str,
        oneof_index: i32,
    ) -> Result<()> {
        let message_struct = format_ident!("{}", message_name.to_upper_camel_case());
        
        for field in &message.field {
            if field.oneof_index == Some(oneof_index) {
                if let prost_types::field_descriptor_proto::Type::Message = field.r#type() {
                    self.generate_constructor_for_field(
                        code_gen,
                        &message_struct,
                        message_name,
                        oneof_name,
                        field,
                    )?;
                }
            }
        }
        
        Ok(())
    }
    
    fn generate_constructor_for_field(
        &self,
        code_gen: &mut CodeGenerator,
        message_struct: &proc_macro2::Ident,
        message_name: &str,
        oneof_name: &str,
        field: &FieldDescriptorProto,
    ) -> Result<()> {
        let field_name = field.name.as_deref().unwrap_or("");
        let field_type_name = field.type_name.as_deref().unwrap_or("");
        
        let variant_name = format_ident!("{}", field_name.to_upper_camel_case());
        let oneof_field = format_ident!("{}", oneof_name.to_snake_case());
        let function_name = format_ident!(
            "new_{}_{}",
            message_name.to_snake_case(),
            field_name.to_snake_case()
        );
        
        let inner_type = self.resolve_type_name(field_type_name);
        let inner_type_ident = format_ident!("{}", inner_type);
        
        let params = self.extract_message_fields(field_type_name);
        let param_declarations: Vec<_> = params
            .iter()
            .map(|(name, ty)| {
                let name_ident = format_ident!("{}", name);
                quote! { #name_ident: #ty }
            })
            .collect();
        
        let field_assignments: Vec<_> = params
            .iter()
            .map(|(name, _)| {
                let field_name = format_ident!("{}", name);
                quote! { #field_name }
            })
            .collect();
        
        let constructor = quote! {
            pub fn #function_name(#(#param_declarations),*) -> #message_struct {
                #message_struct {
                    #oneof_field: Some(#message_struct::#variant_name(#inner_type_ident {
                        #(#field_assignments),*
                    })),
                    ..Default::default()
                }
            }
        };
        
        code_gen.add_item(constructor);
        Ok(())
    }
    
    fn resolve_type_name(&self, type_name: &str) -> String {
        type_name
            .split('.')
            .last()
            .unwrap_or(type_name)
            .to_upper_camel_case()
    }
    
    fn extract_message_fields(&self, type_name: &str) -> Vec<(String, String)> {
        for message in &self.file.message_type {
            if let Some(msg_name) = &message.name {
                if type_name.ends_with(msg_name) {
                    return message
                        .field
                        .iter()
                        .filter_map(|f| {
                            f.name.as_ref().map(|name| {
                                let rust_type = self.field_to_rust_type(f);
                                (name.to_snake_case(), rust_type)
                            })
                        })
                        .collect();
                }
            }
        }
        Vec::new()
    }
    
    fn field_to_rust_type(&self, field: &FieldDescriptorProto) -> String {
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
            Type::Message | Type::Enum => {
                self.resolve_type_name(field.type_name.as_deref().unwrap_or("Unknown"))
            }
            Type::Group => "Unknown".to_string(),
        };
        
        match field.label() {
            Label::Optional if field.proto3_optional.unwrap_or(false) => {
                format!("Option<{}>", base_type)
            }
            Label::Repeated => format!("Vec<{}>", base_type),
            _ => base_type,
        }
    }
}