use quote::quote;
use proc_macro2::TokenStream;
use prettyplease::unparse;

#[cfg(test)]
mod tests;

pub struct CodeGenerator {
    imports: Vec<TokenStream>,
    items: Vec<TokenStream>,
}

impl CodeGenerator {
    pub fn new() -> Self {
        Self {
            imports: Vec::new(),
            items: Vec::new(),
        }
    }
    
    pub fn add_import(&mut self, import: TokenStream) {
        self.imports.push(import);
    }
    
    pub fn add_item(&mut self, item: TokenStream) {
        self.items.push(item);
    }
    
    pub fn generate(self) -> String {
        let imports = self.imports;
        let items = self.items;
        
        let file = quote! {
            #(#imports)*
            
            #(#items)*
        };
        
        let file_str = file.to_string();
        
        match syn::parse_file(&file_str) {
            Ok(syntax_tree) => unparse(&syntax_tree),
            Err(_) => file_str,
        }
    }
    
    pub fn generate_function(
        name: &str,
        params: Vec<(&str, &str)>,
        return_type: Option<&str>,
        body: TokenStream,
    ) -> TokenStream {
        let fn_name = syn::Ident::new(name, proc_macro2::Span::call_site());
        let params: Vec<TokenStream> = params
            .into_iter()
            .map(|(name, ty)| {
                let name = syn::Ident::new(name, proc_macro2::Span::call_site());
                let ty: syn::Type = syn::parse_str(ty).unwrap();
                quote! { #name: #ty }
            })
            .collect();
        
        let return_type = return_type.map(|ty| {
            let ty: syn::Type = syn::parse_str(ty).unwrap();
            quote! { -> #ty }
        });
        
        quote! {
            pub fn #fn_name(#(#params),*) #return_type {
                #body
            }
        }
    }
    
    pub fn generate_struct(
        name: &str,
        fields: Vec<(&str, &str, bool)>,
        derives: Vec<&str>,
    ) -> TokenStream {
        let struct_name = syn::Ident::new(name, proc_macro2::Span::call_site());
        let derives: Vec<syn::Path> = derives.iter().map(|d| syn::parse_str(d).unwrap()).collect();
        
        let fields: Vec<TokenStream> = fields
            .into_iter()
            .map(|(name, ty, is_pub)| {
                let field_name = syn::Ident::new(name, proc_macro2::Span::call_site());
                let field_type: syn::Type = syn::parse_str(ty).unwrap();
                if is_pub {
                    quote! { pub #field_name: #field_type }
                } else {
                    quote! { #field_name: #field_type }
                }
            })
            .collect();
        
        quote! {
            #[derive(#(#derives),*)]
            pub struct #struct_name {
                #(#fields),*
            }
        }
    }
}