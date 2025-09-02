use std::io::{Read, Write};
use prost::Message;
use prost_types::compiler::{CodeGeneratorRequest, CodeGeneratorResponse};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum PluginError {
    #[error("Failed to read from stdin: {0}")]
    ReadError(#[from] std::io::Error),
    
    #[error("Failed to decode protobuf request: {0}")]
    DecodeError(#[from] prost::DecodeError),
    
    #[error("Failed to encode protobuf response: {0}")]
    EncodeError(#[from] prost::EncodeError),
    
    #[error("Generation error: {0}")]
    GenerationError(String),
}

pub type PluginResult<T> = Result<T, PluginError>;

pub trait Plugin {
    fn process(&self, request: CodeGeneratorRequest) -> PluginResult<CodeGeneratorResponse>;
}

pub fn run_plugin<P: Plugin>(plugin: P) -> PluginResult<()> {
    let mut input = Vec::new();
    std::io::stdin().read_to_end(&mut input)?;
    
    let request = CodeGeneratorRequest::decode(&input[..])?;
    let response = plugin.process(request)?;
    
    let output = response.encode_to_vec();
    std::io::stdout().write_all(&output)?;
    
    Ok(())
}