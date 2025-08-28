module github.com/SebastienMelki/sebuf/examples/simple-api

go 1.24.4

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.8-20250717185734-6c6e0d3c608e.1
	buf.build/go/protovalidate v0.14.0
	github.com/SebastienMelki/sebuf v0.0.0-20250818125809-ff61bcf670dd
	google.golang.org/protobuf v1.36.8
)

replace github.com/SebastienMelki/sebuf => ../..

require (
	cel.dev/expr v0.23.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/google/cel-go v0.25.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250811230008-5f3141c8851a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)
