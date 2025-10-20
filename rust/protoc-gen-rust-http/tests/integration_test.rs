use std::process::Command;
use tempfile::TempDir;
use std::fs;

const TEST_SERVICE_PROTO: &str = r#"
syntax = "proto3";

package test.api;

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

message GetUserRequest {
  string user_id = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
}

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}

service AuthService {
  rpc Login(CreateUserRequest) returns (User);
}
"#;

#[test]
fn test_http_service_generation() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("service.proto");
    let output_path = temp_dir.path();

    fs::write(&proto_path, TEST_SERVICE_PROTO).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-http");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-http={}", binary_path))
        .arg(&format!("--rust-http_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("service.proto")
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

    // Check that HTTP files were generated for each service
    let user_service_file = output_path.join("user_service.http.rs");
    let auth_service_file = output_path.join("auth_service.http.rs");

    assert!(user_service_file.exists(), "UserService HTTP file was not generated");
    assert!(auth_service_file.exists(), "AuthService HTTP file was not generated");

    // Verify UserService generated content
    let user_content = fs::read_to_string(&user_service_file)
        .expect("Failed to read UserService file");

    // Commented out for cleaner test output
    // println!("Generated UserService content:\n{}", user_content);

    // Check for trait definition
    assert!(user_content.contains("pub trait UserServiceServer"));
    assert!(user_content.contains("async fn create_user"));
    assert!(user_content.contains("async fn get_user"));
    assert!(user_content.contains("async fn list_users"));

    // Check for router function
    assert!(user_content.contains("pub fn register_user_service_server"));

    // Check for handler functions
    assert!(user_content.contains("async fn create_user_handler"));
    assert!(user_content.contains("async fn get_user_handler"));
    assert!(user_content.contains("async fn list_users_handler"));

    // Check for axum imports
    assert!(user_content.contains("use axum"));
    assert!(user_content.contains("extract"));
    assert!(user_content.contains("Json"));
    assert!(user_content.contains("State"));

    // Verify AuthService generated content
    let auth_content = fs::read_to_string(&auth_service_file)
        .expect("Failed to read AuthService file");

    assert!(auth_content.contains("pub trait AuthServiceServer"));
    assert!(auth_content.contains("async fn login"));
    assert!(auth_content.contains("pub fn register_auth_service_server"));
    assert!(auth_content.contains("async fn login_handler"));

    // Check that the generated code is syntactically valid
    let user_syntax = syn::parse_file(&user_content);
    assert!(user_syntax.is_ok(), "Generated UserService code is not valid Rust: {:?}", user_syntax.err());

    let auth_syntax = syn::parse_file(&auth_content);
    assert!(auth_syntax.is_ok(), "Generated AuthService code is not valid Rust: {:?}", auth_syntax.err());
}

#[test]
fn test_no_services_no_output() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("messages.proto");
    let output_path = temp_dir.path();

    let messages_proto = r#"
syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
}

message Product {
  string id = 1;
  string name = 2;
  double price = 3;
}
"#;

    fs::write(&proto_path, messages_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-http");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-http={}", binary_path))
        .arg(&format!("--rust-http_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("messages.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success(), "protoc should succeed even with no services");

    // Should not generate any .http.rs files
    let entries: Vec<_> = fs::read_dir(output_path)
        .expect("Failed to read output directory")
        .collect();

    let http_files: Vec<_> = entries
        .into_iter()
        .filter_map(|entry| entry.ok())
        .filter(|entry| {
            entry.file_name().to_string_lossy().ends_with(".http.rs")
        })
        .collect();

    assert!(http_files.is_empty(), "No HTTP files should be generated for message-only protos");
}

#[test]
fn test_service_with_no_methods() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("empty_service.proto");
    let output_path = temp_dir.path();

    let empty_service_proto = r#"
syntax = "proto3";

package test;

service EmptyService {
  // No methods defined
}
"#;

    fs::write(&proto_path, empty_service_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-http");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-http={}", binary_path))
        .arg(&format!("--rust-http_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("empty_service.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success(), "protoc should succeed with empty service");

    let service_file = output_path.join("empty_service.http.rs");
    if service_file.exists() {
        let content = fs::read_to_string(&service_file)
            .expect("Failed to read service file");
        
        // Should still generate trait and router, just empty
        assert!(content.contains("pub trait EmptyServiceServer"));
        assert!(content.contains("pub fn register_empty_service_server"));
    }
}