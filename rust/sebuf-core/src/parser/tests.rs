#[cfg(test)]
mod tests {
    use super::super::*;
    use prost_types::{
        DescriptorProto, FieldDescriptorProto, FileDescriptorProto, ServiceDescriptorProto,
        field_descriptor_proto::{Label, Type},
    };

    fn create_test_field(name: &str, field_type: Type, label: Label) -> FieldDescriptorProto {
        FieldDescriptorProto {
            name: Some(name.to_string()),
            r#type: Some(field_type as i32),
            label: Some(label as i32),
            type_name: None,
            ..Default::default()
        }
    }

    fn create_test_message_field(name: &str, type_name: &str, label: Label) -> FieldDescriptorProto {
        FieldDescriptorProto {
            name: Some(name.to_string()),
            r#type: Some(Type::Message as i32),
            label: Some(label as i32),
            type_name: Some(type_name.to_string()),
            ..Default::default()
        }
    }

    #[test]
    fn test_field_to_rust_type_scalars() {
        let cases = vec![
            (Type::Double, Label::Required, "f64"),
            (Type::Float, Label::Required, "f32"),
            (Type::Int64, Label::Required, "i64"),
            (Type::Uint64, Label::Required, "u64"),
            (Type::Int32, Label::Required, "i32"),
            (Type::Bool, Label::Required, "bool"),
            (Type::String, Label::Required, "String"),
            (Type::Bytes, Label::Required, "Vec<u8>"),
        ];

        for (field_type, label, expected) in cases {
            let field = create_test_field("test", field_type, label);
            assert_eq!(TypeMapper::field_to_rust_type(&field), expected);
        }
    }

    #[test]
    fn test_field_to_rust_type_optional() {
        let field = create_test_field("test", Type::String, Label::Optional);
        assert_eq!(TypeMapper::field_to_rust_type(&field), "Option<String>");
    }

    #[test]
    fn test_field_to_rust_type_repeated() {
        let field = create_test_field("test", Type::Int32, Label::Repeated);
        assert_eq!(TypeMapper::field_to_rust_type(&field), "Vec<i32>");
    }

    #[test]
    fn test_field_to_rust_type_message() {
        let field = create_test_message_field("user", ".example.User", Label::Required);
        assert_eq!(TypeMapper::field_to_rust_type(&field), ".example.User");
    }

    #[test]
    fn test_message_name_to_rust() {
        assert_eq!(TypeMapper::message_name_to_rust("user_info"), "UserInfo");
        assert_eq!(TypeMapper::message_name_to_rust("UserInfo"), "UserInfo");
        assert_eq!(TypeMapper::message_name_to_rust("user-info"), "UserInfo");
    }

    #[test]
    fn test_field_name_to_rust() {
        assert_eq!(TypeMapper::field_name_to_rust("UserName"), "user_name");
        assert_eq!(TypeMapper::field_name_to_rust("userName"), "user_name");
        assert_eq!(TypeMapper::field_name_to_rust("user_name"), "user_name");
    }

    #[test]
    fn test_method_name_to_rust() {
        assert_eq!(TypeMapper::method_name_to_rust("GetUser"), "get_user");
        assert_eq!(TypeMapper::method_name_to_rust("CreateUserAccount"), "create_user_account");
    }

    #[test]
    fn test_proto_parser_find_message() {
        let file = FileDescriptorProto {
            name: Some("test.proto".to_string()),
            package: Some("test".to_string()),
            message_type: vec![
                DescriptorProto {
                    name: Some("User".to_string()),
                    ..Default::default()
                },
                DescriptorProto {
                    name: Some("Account".to_string()),
                    ..Default::default()
                },
            ],
            ..Default::default()
        };

        let parser = ProtoParser::new(vec![file]);
        
        assert!(parser.find_message("User").is_some());
        assert!(parser.find_message("Account").is_some());
        assert!(parser.find_message("NonExistent").is_none());
    }

    #[test]
    fn test_proto_parser_find_service() {
        let file = FileDescriptorProto {
            name: Some("test.proto".to_string()),
            package: Some("test".to_string()),
            service: vec![
                ServiceDescriptorProto {
                    name: Some("UserService".to_string()),
                    ..Default::default()
                },
            ],
            ..Default::default()
        };

        let parser = ProtoParser::new(vec![file]);
        
        assert!(parser.find_service("UserService").is_some());
        assert!(parser.find_service("NonExistent").is_none());
    }
}