use anyhow::Result;
use sebuf_core::{run_plugin, Plugin, PluginError, PluginResult};
use prost_types::compiler::{CodeGeneratorRequest, CodeGeneratorResponse};

mod generator;
mod schema;
use generator::OpenApiGenerator;

struct OpenApiPlugin;

impl Plugin for OpenApiPlugin {
    fn process(&self, request: CodeGeneratorRequest) -> PluginResult<CodeGeneratorResponse> {
        let mut response = CodeGeneratorResponse::default();
        
        for proto_file in request.proto_file.iter() {
            if !request.file_to_generate.contains(&proto_file.name.clone().unwrap_or_default()) {
                continue;
            }
            
            if proto_file.service.is_empty() {
                continue;
            }
            
            let generator = OpenApiGenerator::new(proto_file.clone(), &request.proto_file);
            match generator.generate() {
                Ok(generated_files) => {
                    response.file.extend(generated_files);
                }
                Err(e) => {
                    return Err(PluginError::GenerationError(e.to_string()));
                }
            }
        }
        
        response.supported_features = Some(
            prost_types::compiler::code_generator_response::Feature::Proto3Optional as u64
        );
        
        Ok(response)
    }
}

fn main() -> Result<()> {
    run_plugin(OpenApiPlugin)?;
    Ok(())
}