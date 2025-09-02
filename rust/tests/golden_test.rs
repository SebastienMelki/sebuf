use std::path::PathBuf;
use std::process::Command;
use std::fs;
use tempfile::TempDir;

const UPDATE_GOLDEN: &str = "UPDATE_GOLDEN";

#[derive(Debug)]
struct TestCase {
    name: &'static str,
    proto_content: &'static str,
    plugin: &'static str,
    binary_name: &'static str,
    output_extension: &'static str,
}

impl TestCase {
    fn run(&self) {
        println!("Running golden test: {}", self.name);
        
        let temp_dir = TempDir::new().expect("Failed to create temp dir");
        let proto_path = temp_dir.path().join("test.proto");
        let output_path = temp_dir.path();

        // Write test proto file
        fs::write(&proto_path, self.proto_content).expect("Failed to write proto file");

        // Get binary path
        let binary_path = env!(&format!("CARGO_BIN_EXE_{}", self.binary_name));

        // Run protoc with our plugin
        let output = Command::new("protoc")
            .arg(&format!("--plugin={}={}", self.plugin, binary_path))
            .arg(&format!("--{}_out={}", self.plugin.trim_start_matches("protoc-gen-"), output_path.display()))
            .arg(&format!("--proto_path={}", temp_dir.path().display()))
            .arg("test.proto")
            .output()
            .expect("Failed to execute protoc");

        if !output.status.success() {
            panic!(
                "protoc failed for test {}: {}\nstdout: {}\nstderr: {}",
                self.name,
                output.status,
                String::from_utf8_lossy(&output.stdout),
                String::from_utf8_lossy(&output.stderr)
            );
        }

        // Find generated files
        let generated_files: Vec<_> = fs::read_dir(output_path)
            .expect("Failed to read output directory")
            .filter_map(|entry| entry.ok())
            .filter(|entry| {
                let name = entry.file_name();
                let name_str = name.to_string_lossy();
                name_str.ends_with(self.output_extension) && 
                !name_str.starts_with('.') &&
                name_str != "test.proto"
            })
            .collect();

        if generated_files.is_empty() {
            // Some tests might not generate files (e.g., no oneofs), that's OK
            return;
        }

        // Process each generated file
        for file_entry in generated_files {
            let file_path = file_entry.path();
            let file_name = file_entry.file_name();
            let content = fs::read_to_string(&file_path)
                .expect("Failed to read generated file");

            let golden_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
                .parent().unwrap()
                .join("testdata/golden")
                .join(match self.plugin.as_ref() {
                    "protoc-gen-rust-oneof-helper" => "oneof-helper",
                    "protoc-gen-rust-http" => "http", 
                    "protoc-gen-rust-openapiv3" => "openapi",
                    _ => panic!("Unknown plugin: {}", self.plugin),
                });

            fs::create_dir_all(&golden_dir).expect("Failed to create golden directory");

            let golden_file = golden_dir.join(format!("{}_{}", self.name, file_name.to_string_lossy()));

            if std::env::var(UPDATE_GOLDEN).is_ok() {
                // Update golden file
                fs::write(&golden_file, &content).expect("Failed to write golden file");
                println!("Updated golden file: {:?}", golden_file);
            } else {
                // Compare with golden file
                if !golden_file.exists() {
                    panic!(
                        "Golden file does not exist: {:?}\n\
                        Run with UPDATE_GOLDEN=1 to create it.\n\
                        Generated content:\n{}",
                        golden_file, content
                    );
                }

                let golden_content = fs::read_to_string(&golden_file)
                    .expect("Failed to read golden file");

                if content != golden_content {
                    panic!(
                        "Generated content differs from golden file: {:?}\n\
                        Run with UPDATE_GOLDEN=1 to update.\n\
                        \nExpected:\n{}\n\nActual:\n{}",
                        golden_file, golden_content, content
                    );
                }
            }
        }
    }
}

static TEST_CASES: &[TestCase] = &[
    TestCase {
        name: "simple_oneof",
        proto_content: r#"
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
"#,
        plugin: "protoc-gen-rust-oneof-helper",
        binary_name: "protoc-gen-rust-oneof-helper",
        output_extension: ".oneof_helpers.rs",
    },

    TestCase {
        name: "simple_service",
        proto_content: r#"
syntax = "proto3";
package test;

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(CreateUserRequest) returns (User);
}
"#,
        plugin: "protoc-gen-rust-http",
        binary_name: "protoc-gen-rust-http",
        output_extension: ".http.rs",
    },

    TestCase {
        name: "simple_openapi",
        proto_content: r#"
syntax = "proto3";
package test;

message User {
  string id = 1;
  string name = 2;
}

message GetUserRequest {
  string user_id = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (User);
}
"#,
        plugin: "protoc-gen-rust-openapiv3",
        binary_name: "protoc-gen-rust-openapiv3", 
        output_extension: ".openapi.yaml",
    },

    TestCase {
        name: "complex_types",
        proto_content: r#"
syntax = "proto3";
package test;

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

message User {
  string id = 1;
  string name = 2;
  repeated string tags = 3;
  map<string, string> metadata = 4;
  Status status = 5;
  optional string phone = 6;
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
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
"#,
        plugin: "protoc-gen-rust-openapiv3",
        binary_name: "protoc-gen-rust-openapiv3",
        output_extension: ".openapi.yaml",
    },
];

#[test]
fn test_golden_files() {
    for test_case in TEST_CASES {
        test_case.run();
    }
}