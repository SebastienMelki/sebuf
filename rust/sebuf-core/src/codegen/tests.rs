#[cfg(test)]
mod tests {
    use super::super::*;

    #[test]
    fn test_code_generator_new() {
        let gen = CodeGenerator::new();
        assert!(gen.imports.is_empty());
        assert!(gen.items.is_empty());
    }

    #[test]
    fn test_generate_function() {
        let func = CodeGenerator::generate_function(
            "hello_world",
            vec![("name", "String"), ("age", "i32")],
            Some("String"),
            quote! { format!("Hello, {}! Age: {}", name, age) }
        );

        let generated = func.to_string();
        assert!(generated.contains("pub fn hello_world"));
        assert!(generated.contains("name :"));
        assert!(generated.contains("String"));
        assert!(generated.contains("age :"));
        assert!(generated.contains("i32"));
        assert!(generated.contains("-> String"));
    }

    #[test]
    fn test_generate_function_no_return() {
        let func = CodeGenerator::generate_function(
            "print_hello",
            vec![("name", "&str")],
            None,
            quote! { println!("Hello, {}", name); }
        );

        let generated = func.to_string();
        assert!(generated.contains("pub fn print_hello"));
        assert!(generated.contains("name :"));
        assert!(generated.contains("& str"));
        assert!(!generated.contains(" -> "));
    }

    #[test]
    fn test_generate_struct() {
        let struct_def = CodeGenerator::generate_struct(
            "User",
            vec![
                ("id", "String", true),
                ("name", "String", true),
                ("age", "i32", false),
            ],
            vec!["Debug", "Clone"],
        );

        let generated = struct_def.to_string();
        assert!(generated.contains("derive (Debug"));
        assert!(generated.contains("Clone)"));
        assert!(generated.contains("pub struct User"));
        assert!(generated.contains("pub id : String"));
        assert!(generated.contains("pub name : String"));
        assert!(generated.contains("age : i32"));
    }

    #[test]
    fn test_code_generator_with_imports_and_items() {
        let mut gen = CodeGenerator::new();
        
        gen.add_import(quote! { use std::collections::HashMap; });
        gen.add_import(quote! { use serde::{Serialize, Deserialize}; });
        
        gen.add_item(quote! {
            #[derive(Debug)]
            pub struct TestStruct {
                pub data: HashMap<String, String>,
            }
        });
        
        let generated = gen.generate();
        
        assert!(generated.contains("use std::collections::HashMap"));
        assert!(generated.contains("use serde::{Serialize, Deserialize}"));
        assert!(generated.contains("pub struct TestStruct"));
        assert!(generated.contains("HashMap<String, String>"));
    }
}