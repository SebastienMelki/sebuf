// Package annotations provides shared annotation parsing for all sebuf protoc plugins.
//
// This package extracts HTTP configuration, headers, query parameters, unwrap annotations,
// field examples, path parameters, and helper utilities from protobuf definitions. All four
// generators (httpgen, clientgen, tsclientgen, openapiv3) import this package instead of
// maintaining their own duplicated annotation parsing code.
//
// # Convention-based extensibility
//
// Each annotation concept lives in its own file with standardized function signatures:
//
//   - http_config.go:    GetMethodHTTPConfig, GetServiceBasePath
//   - headers.go:        GetServiceHeaders, GetMethodHeaders, CombineHeaders
//   - query.go:          GetQueryParams
//   - unwrap.go:         HasUnwrapAnnotation, GetUnwrapField, FindUnwrapField, IsRootUnwrap
//   - field_examples.go: GetFieldExamples
//   - path.go:           ExtractPathParams, BuildHTTPPath, EnsureLeadingSlash
//   - method.go:         HTTPMethodToString, HTTPMethodToLower
//   - helpers.go:        LowerFirst
//
// To add a new annotation type, create a new file following this pattern:
//
//  1. Define any needed structs with exported fields.
//  2. Add GetXxx() or ParseXxx() functions that accept protogen types.
//  3. Use proto.GetExtension to extract the annotation from options.
package annotations
