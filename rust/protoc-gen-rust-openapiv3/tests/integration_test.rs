use std::process::Command;
use tempfile::TempDir;
use std::fs;
use serde_yaml;
use serde_json;

const TEST_API_PROTO: &str = r#"
syntax = "proto3";

package test.api.v1;

enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  UserStatus status = 4;
  repeated string tags = 5;
  map<string, string> metadata = 6;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  UserStatus initial_status = 3;
}

message GetUserRequest {
  string user_id = 1;
}

message UpdateUserRequest {
  string user_id = 1;
  optional string name = 2;
  optional string email = 3;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);  
  rpc UpdateUser(UpdateUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}

service AdminService {
  rpc DeleteUser(GetUserRequest) returns (User);
}
"#;

#[test]
fn test_openapi_generation() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("api.proto");
    let output_path = temp_dir.path();

    fs::write(&proto_path, TEST_API_PROTO).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-openapiv3");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-openapiv3={}", binary_path))
        .arg(&format!("--rust-openapiv3_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("api.proto")
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

    // Check that OpenAPI files were generated for each service
    let user_service_file = output_path.join("UserService.openapi.yaml");
    let admin_service_file = output_path.join("AdminService.openapi.yaml");

    assert!(user_service_file.exists(), "UserService OpenAPI file was not generated");
    assert!(admin_service_file.exists(), "AdminService OpenAPI file was not generated");

    // Parse and verify UserService OpenAPI spec
    let user_content = fs::read_to_string(&user_service_file)
        .expect("Failed to read UserService OpenAPI file");

    let user_spec: serde_yaml::Value = serde_yaml::from_str(&user_content)
        .expect("Generated OpenAPI is not valid YAML");

    // Verify basic OpenAPI structure
    assert_eq!(user_spec["openapi"], "3.1.0");
    assert_eq!(user_spec["info"]["title"], "UserService API");
    assert_eq!(user_spec["info"]["version"], "1.0.0");

    // Verify paths are generated
    let paths = user_spec["paths"].as_mapping().expect("paths should be an object");
    assert!(!paths.is_empty(), "Should have generated paths");

    // Check for expected paths (using default path pattern)
    let expected_paths = vec![
        "/api/v1/create_user",
        "/api/v1/get_user", 
        "/api/v1/update_user",
        "/api/v1/list_users",
    ];

    for expected_path in expected_paths {
        assert!(paths.contains_key(&serde_yaml::Value::String(expected_path.to_string())),
               "Missing expected path: {}", expected_path);
    }

    // Verify components/schemas are generated
    let components = user_spec["components"].as_mapping().expect("components should exist");
    let schemas = components["schemas"].as_mapping().expect("schemas should exist");

    let expected_schemas = vec![
        "User",
        "CreateUserRequest", 
        "GetUserRequest",
        "UpdateUserRequest",
        "ListUsersRequest",
        "ListUsersResponse",
        "UserStatus",
    ];

    for expected_schema in expected_schemas {
        assert!(schemas.contains_key(&serde_yaml::Value::String(expected_schema.to_string())),
               "Missing expected schema: {}", expected_schema);
    }

    // Verify User schema structure
    let user_schema = &schemas["User"];
    assert_eq!(user_schema["type"], "object");
    
    let user_properties = user_schema["properties"].as_mapping().expect("User should have properties");
    assert!(user_properties.contains_key("id"));
    assert!(user_properties.contains_key("name"));
    assert!(user_properties.contains_key("email"));
    assert!(user_properties.contains_key("status"));
    assert!(user_properties.contains_key("tags"));
    assert!(user_properties.contains_key("metadata"));

    // Verify enum schema
    let status_schema = &schemas["UserStatus"];
    assert_eq!(status_schema["type"], "string");
    let enum_values = status_schema["enum"].as_sequence().expect("enum should have values");
    assert!(enum_values.len() >= 3); // Should have the defined enum values

    // Verify array type (tags field)
    let tags_prop = &user_properties["tags"];
    assert_eq!(tags_prop["type"], "array");
    assert!(tags_prop["items"].is_mapping());

    // Parse AdminService spec
    let admin_content = fs::read_to_string(&admin_service_file)
        .expect("Failed to read AdminService OpenAPI file");

    let admin_spec: serde_yaml::Value = serde_yaml::from_str(&admin_content)
        .expect("AdminService OpenAPI is not valid YAML");

    assert_eq!(admin_spec["info"]["title"], "AdminService API");
    
    let admin_paths = admin_spec["paths"].as_mapping().expect("admin paths should exist");
    assert!(admin_paths.contains_key("/api/v1/delete_user"));
}

#[test]
fn test_messages_only_no_output() {
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
"#;

    fs::write(&proto_path, messages_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-openapiv3");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-openapiv3={}", binary_path))
        .arg(&format!("--rust-openapiv3_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("messages.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success(), "protoc should succeed with messages-only proto");

    // Should not generate any OpenAPI files
    let entries: Vec<_> = fs::read_dir(output_path)
        .expect("Failed to read output directory")
        .collect();

    let openapi_files: Vec<_> = entries
        .into_iter()
        .filter_map(|entry| entry.ok())
        .filter(|entry| {
            entry.file_name().to_string_lossy().ends_with(".openapi.yaml")
        })
        .collect();

    assert!(openapi_files.is_empty(), "No OpenAPI files should be generated for message-only protos");
}

#[test]
fn test_openapi_spec_validates_json_schema() {
    let temp_dir = TempDir::new().expect("Failed to create temp dir");
    let proto_path = temp_dir.path().join("simple.proto");
    let output_path = temp_dir.path();

    let simple_proto = r#"
syntax = "proto3";

package test;

message SimpleMessage {
  string text = 1;
  int32 number = 2;
  bool flag = 3;
}

service SimpleService {
  rpc DoSomething(SimpleMessage) returns (SimpleMessage);
}
"#;

    fs::write(&proto_path, simple_proto).expect("Failed to write proto file");

    let binary_path = env!("CARGO_BIN_EXE_protoc-gen-rust-openapiv3");

    let output = Command::new("protoc")
        .arg(&format!("--plugin=protoc-gen-rust-openapiv3={}", binary_path))
        .arg(&format!("--rust-openapiv3_out={}", output_path.display()))
        .arg(&format!("--proto_path={}", temp_dir.path().display()))
        .arg("simple.proto")
        .output()
        .expect("Failed to execute protoc");

    assert!(output.status.success());

    let spec_file = output_path.join("SimpleService.openapi.yaml");
    assert!(spec_file.exists());

    let content = fs::read_to_string(&spec_file).expect("Failed to read spec file");
    let spec: serde_yaml::Value = serde_yaml::from_str(&content)
        .expect("Should be valid YAML");

    // Convert to JSON to verify it's valid JSON Schema-compatible
    let json_str = serde_json::to_string_pretty(&spec).expect("Should convert to JSON");
    let _json_value: serde_json::Value = serde_json::from_str(&json_str)
        .expect("Should be valid JSON");

    // Verify the SimpleMessage schema has correct types
    let schemas = &spec["components"]["schemas"];
    let simple_msg = &schemas["SimpleMessage"];
    let properties = simple_msg["properties"].as_mapping().unwrap();
    
    assert_eq!(properties["text"]["type"], "string");
    assert_eq!(properties["number"]["type"], "integer");
    assert_eq!(properties["flag"]["type"], "boolean");
}