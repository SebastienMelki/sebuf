use std::process::Command;
use tempfile::TempDir;
use std::fs;
use std::path::Path;

const TEST_PROTO: &str = r#"
syntax = "proto3";

package test;

message LoginRequest {
  oneof auth_method {
    EmailAuth email = 1;
    PhoneAuth phone = 2;
  }
  
  message EmailAuth {
    string email = 1;
    string password = 2;
  }
  
  message PhoneAuth {
    string phone = 1;
    string code = 2;
  }
}

message PaymentMethod {
  oneof method {
    CreditCard card = 1;
    BankAccount bank = 2;
  }
  
  message CreditCard {
    string number = 1;
    string expiry = 2;
    string cvv = 3;
  }
  
  message BankAccount {
    string account_number = 1;
    string routing_number = 2;
  }
}
"#;

#[test]
fn test_oneof_helper_generation() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("test.proto");
    let output_path = temp_dir.path();

    // Write test proto file
    fs::write(&proto_path, TEST_PROTO).expect("Failed to write proto file");

    // Get the binary path
    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-oneof-helper");

    // Run protoc with our plugin
    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-oneof-helper={}", binary_path))
        .arg(&format!("--rust-oneof-helper_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("test.proto")
        .output()
        .expect("Failed to execute protoc");

    if !output.status.success() {
        panic!(
            "protoc failed: {}\nstdout: {}\nstderr: {}",
            output.status,
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
    }

    // Check that the helper file was generated
    let helper_file = output_path.join("test.oneof_helpers.rs");
    assert!(helper_file.exists(), "Helper file was not generated");

    // Read and verify the generated content
    let generated_content = fs::read_to_string(&helper_file)
        .expect("Failed to read generated file");

    // Check that some oneof helper functions were generated
    // (specific function names may vary based on implementation)
    assert!(generated_content.contains("pub fn new_"));
    assert!(generated_content.contains("LoginRequest"));
    
    // Should contain basic struct construction
    assert!(generated_content.contains("auth_method"));
    assert!(generated_content.contains("method"));

    // Check that the generated code is syntactically valid Rust
    let syntax_check = syn::parse_file(&generated_content);
    assert!(syntax_check.is_ok(), "Generated code is not valid Rust syntax");
}

#[test]
fn test_no_oneofs_no_output() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("simple.proto");
    let output_path = temp_dir.path();

    let simple_proto = r#"
syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
}
"#;

    // Write proto file without oneofs
    fs::write(&proto_path, simple_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-oneof-helper");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-oneof-helper={}", binary_path))
        .arg(&format!("--rust-oneof-helper_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("simple.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success(), "protoc should succeed even with no oneofs");

    // Check that no helper file was generated (or it's empty)
    let helper_file = output_path.join("simple.oneof_helpers.rs");
    if helper_file.exists() {
        let content = fs::read_to_string(&helper_file)
            .expect("Failed to read generated file");
        // Should be empty or minimal
        assert!(content.trim().is_empty() || content.lines().count() < 5);
    }
}

#[test] 
fn test_nested_message_oneof() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("nested.proto");
    let output_path = temp_dir.path();

    let nested_proto = r#"
syntax = "proto3";

package test;

message Container {
  message InnerMessage {
    oneof value {
      StringValue str = 1;
      IntValue int = 2;
    }
    
    message StringValue {
      string data = 1;
    }
    
    message IntValue {
      int32 data = 1;
    }
  }
}
"#;

    fs::write(&proto_path, nested_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-oneof-helper");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-oneof-helper={}", binary_path))
        .arg(&format!("--rust-oneof-helper_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("nested.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success(), "protoc should handle nested messages");

    let helper_file = output_path.join("nested.oneof_helpers.rs");
    if helper_file.exists() {
        let generated_content = fs::read_to_string(&helper_file)
            .expect("Failed to read generated file");
        
        // Should generate helpers for nested oneof
        assert!(generated_content.contains("new_inner_message_str") ||
                generated_content.contains("new_inner_message_int"));
    }
}