use anyhow::Result;
use sebuf_core::{run_plugin, Plugin, PluginError, PluginResult};
use prost_types::compiler::{CodeGeneratorRequest, CodeGeneratorResponse};

mod generator;
use generator::OneofHelperGenerator;

struct OneofHelperPlugin;

impl Plugin for OneofHelperPlugin {
    fn process(&self, request: CodeGeneratorRequest) -> PluginResult<CodeGeneratorResponse> {
        let mut response = CodeGeneratorResponse::default();
        
        for proto_file in request.proto_file {
            if !request.file_to_generate.contains(&proto_file.name.clone().unwrap_or_default()) {
                continue;
            }
            
            let generator = OneofHelperGenerator::new(proto_file.clone());
            match generator.generate() {
                Ok(Some(generated)) => {
                    response.file.push(generated);
                }
                Ok(None) => {}
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
    run_plugin(OneofHelperPlugin)?;
    Ok(())
}