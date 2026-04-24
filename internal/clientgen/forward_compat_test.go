package clientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestForwardCompatGoldenPatterns verifies the generated code has the correct
// structural patterns for the sebufUnmarshaler interface and discard-unknown-fields support.
func TestForwardCompatGoldenPatterns(t *testing.T) {
	goldenDir := filepath.Join("testdata", "golden")

	// K.36 — Golden assertion: every annotated message gets UnmarshalJSONSebuf
	t.Run("K36_annotated_messages_have_UnmarshalJSONSebuf", func(t *testing.T) {
		encodingFiles := []struct {
			file     string
			msgNames []string
		}{
			{"int64_encoding_encoding.pb.go", []string{"Int64EncodingTest"}},
			{"int64_nested_encoding_encoding.pb.go", []string{"SensorReading", "GetSensorReadingResponse", "GetMultiSensorResponse"}},
			{"nullable_nullable.pb.go", []string{"User"}},
			{"timestamp_format_timestamp_format.pb.go", []string{"TimestampFormatTest"}},
			{"bytes_encoding_bytes_encoding.pb.go", []string{"BytesEncodingTest"}},
			{"empty_behavior_empty_behavior.pb.go", []string{"Response"}},
			{"flatten_flatten.pb.go", []string{"SimpleFlatten", "DualFlatten", "MixedFlatten"}},
			{"oneof_discriminator_oneof_discriminator.pb.go", []string{"FlattenedEvent", "NestedEvent"}},
		}

		for _, ef := range encodingFiles {
			content, err := os.ReadFile(filepath.Join(goldenDir, ef.file))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", ef.file, err)
			}
			s := string(content)
			for _, msg := range ef.msgNames {
				pattern := "func (x *" + msg + ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {"
				if !strings.Contains(s, pattern) {
					t.Errorf("%s: missing UnmarshalJSONSebuf for %s", ef.file, msg)
				}
			}
		}
	})

	// K.37 — Golden assertion: UnmarshalJSON delegates to UnmarshalJSONSebuf
	t.Run("K37_UnmarshalJSON_delegates_to_UnmarshalJSONSebuf", func(t *testing.T) {
		encodingFiles := []struct {
			file     string
			msgNames []string
		}{
			{"int64_encoding_encoding.pb.go", []string{"Int64EncodingTest"}},
			{"nullable_nullable.pb.go", []string{"User"}},
			{"timestamp_format_timestamp_format.pb.go", []string{"TimestampFormatTest"}},
			{"bytes_encoding_bytes_encoding.pb.go", []string{"BytesEncodingTest"}},
			{"empty_behavior_empty_behavior.pb.go", []string{"Response"}},
			{"flatten_flatten.pb.go", []string{"SimpleFlatten", "DualFlatten", "MixedFlatten"}},
			{"oneof_discriminator_oneof_discriminator.pb.go", []string{"FlattenedEvent", "NestedEvent"}},
			{"int64_nested_encoding_encoding.pb.go", []string{"SensorReading", "GetSensorReadingResponse", "GetMultiSensorResponse"}},
		}

		for _, ef := range encodingFiles {
			content, err := os.ReadFile(filepath.Join(goldenDir, ef.file))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", ef.file, err)
			}
			s := string(content)
			for _, msg := range ef.msgNames {
				pattern := "return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})"
				wrapperSig := "func (x *" + msg + ") UnmarshalJSON(data []byte) error {"
				if !strings.Contains(s, wrapperSig) {
					t.Errorf("%s: missing UnmarshalJSON wrapper for %s", ef.file, msg)
				}
				if !strings.Contains(s, pattern) {
					t.Errorf("%s: UnmarshalJSON doesn't delegate to UnmarshalJSONSebuf for %s", ef.file, msg)
				}
			}
		}
	})

	// Verify sebufUnmarshaler interface is defined in client files
	t.Run("sebufUnmarshaler_interface_defined", func(t *testing.T) {
		clientFiles := []string{
			"backward_compat_client.pb.go",
			"int64_encoding_client.pb.go",
			"nullable_client.pb.go",
			"flatten_client.pb.go",
			"sse_client.pb.go",
		}

		for _, f := range clientFiles {
			content, err := os.ReadFile(filepath.Join(goldenDir, f))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", f, err)
			}
			s := string(content)
			if !strings.Contains(s, "type sebufUnmarshaler interface {") {
				t.Errorf("%s: missing sebufUnmarshaler interface", f)
			}
			if !strings.Contains(s, "UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error") {
				t.Errorf("%s: sebufUnmarshaler missing UnmarshalJSONSebuf method", f)
			}
		}
	})

	// Verify unmarshalResponse checks sebufUnmarshaler first
	t.Run("unmarshalResponse_checks_sebufUnmarshaler_first", func(t *testing.T) {
		clientFiles := []string{
			"backward_compat_client.pb.go",
			"int64_encoding_client.pb.go",
			"sse_client.pb.go",
		}

		for _, f := range clientFiles {
			content, err := os.ReadFile(filepath.Join(goldenDir, f))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", f, err)
			}
			s := string(content)

			// Verify signature has discardUnknown parameter
			if !strings.Contains(s, "unmarshalResponse(body []byte, msg proto.Message, contentType string, discardUnknown bool) error") {
				t.Errorf("%s: unmarshalResponse missing discardUnknown parameter", f)
			}
			// Verify opts built from discardUnknown
			if !strings.Contains(s, "opts := protojson.UnmarshalOptions{DiscardUnknown: discardUnknown}") {
				t.Errorf("%s: unmarshalResponse not building opts from discardUnknown", f)
			}
			// Verify sebufUnmarshaler checked first
			if !strings.Contains(s, "if u, ok := msg.(sebufUnmarshaler); ok {") {
				t.Errorf("%s: unmarshalResponse not checking sebufUnmarshaler", f)
			}
			if !strings.Contains(s, "return u.UnmarshalJSONSebuf(body, opts)") {
				t.Errorf("%s: unmarshalResponse not calling UnmarshalJSONSebuf", f)
			}
		}
	})

	// Verify client struct has discardUnknownFields field
	t.Run("client_struct_has_discardUnknownFields", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "discardUnknownFields bool") {
			t.Error("client struct missing discardUnknownFields field")
		}
	})

	// Verify call options struct has *bool for per-call override
	t.Run("call_options_has_pointer_bool", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "discardUnknownFields *bool") {
			t.Error("call options missing discardUnknownFields *bool field")
		}
	})

	// Verify WithXxxDiscardUnknownFields option generated
	t.Run("client_option_generated", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "func WithNoAnnotationsServiceDiscardUnknownFields(discard bool)") {
			t.Error("missing WithNoAnnotationsServiceDiscardUnknownFields option")
		}
	})

	// Verify WithXxxCallDiscardUnknownFields option generated with pointer semantics
	t.Run("call_option_generated_with_pointer", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "func WithNoAnnotationsServiceCallDiscardUnknownFields(discard bool)") {
			t.Error("missing WithNoAnnotationsServiceCallDiscardUnknownFields option")
		}
		if !strings.Contains(s, "o.discardUnknownFields = &discard") {
			t.Error("call option not using pointer assignment")
		}
	})

	// Verify discardUnknown resolution in RPC methods
	t.Run("rpc_method_resolves_discard_option", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "discardUnknown := c.discardUnknownFields") {
			t.Error("RPC method not reading client-level discardUnknownFields")
		}
		if !strings.Contains(s, "if callOpts.discardUnknownFields != nil {") {
			t.Error("RPC method not checking per-call override")
		}
		if !strings.Contains(s, "discardUnknown = *callOpts.discardUnknownFields") {
			t.Error("RPC method not applying per-call override")
		}
	})

	// G.29 — handleErrorResponse always uses strict mode (false)
	t.Run("G29_handleErrorResponse_strict_mode", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "backward_compat_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "c.unmarshalResponse(body, validationErr, contentType, false)") {
			t.Error("handleErrorResponse not passing false (strict) for ValidationError")
		}
		if !strings.Contains(s, "c.unmarshalResponse(body, genericErr, contentType, false)") {
			t.Error("handleErrorResponse not passing false (strict) for Error")
		}
		if !strings.Contains(s, "Always use strict mode (false) for error parsing") {
			t.Error("handleErrorResponse missing comment explaining strict mode")
		}
	})

	// SSE EventStream tests
	t.Run("SSE_EventStream_has_discardUnknownFields", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "sse_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		// Verify EventStream struct has discardUnknownFields
		if !strings.Contains(s, "discardUnknownFields bool") {
			t.Error("EventStream missing discardUnknownFields field")
		}

		// Verify SSE method resolves and passes discardUnknown
		if !strings.Contains(s, "discardUnknownFields: discardUnknown,") {
			t.Error("SSE method not passing discardUnknown to EventStream")
		}
	})

	// Verify SSE Next() method uses sebufUnmarshaler
	t.Run("SSE_Next_uses_sebufUnmarshaler", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "sse_client.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "opts := protojson.UnmarshalOptions{DiscardUnknown: s.discardUnknownFields}") {
			t.Error("EventStream.Next not building opts from discardUnknownFields")
		}
		if !strings.Contains(s, "if u, ok := any(event).(sebufUnmarshaler); ok {") {
			t.Error("EventStream.Next not checking sebufUnmarshaler")
		}
		if !strings.Contains(s, "unmarshalErr = u.UnmarshalJSONSebuf([]byte(data), opts)") {
			t.Error("EventStream.Next not calling UnmarshalJSONSebuf")
		}
	})

	// Verify wrapper types forward opts to nested children
	t.Run("wrapper_forwards_opts_to_children", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "int64_nested_encoding_encoding.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error") {
			t.Error("wrapper not checking child for UnmarshalJSONSebuf")
		}
		if !strings.Contains(s, "u.UnmarshalJSONSebuf(rawVal, opts)") {
			t.Error("wrapper not forwarding opts to child UnmarshalJSONSebuf")
		}
	})

	// Verify flatten forwards opts to children
	t.Run("flatten_forwards_opts_to_children", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "flatten_flatten.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error") {
			t.Error("flatten not checking child for UnmarshalJSONSebuf")
		}
	})

	// Verify oneof_discriminator forwards opts to children
	t.Run("oneof_discriminator_forwards_opts_to_children", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "oneof_discriminator_oneof_discriminator.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if !strings.Contains(s, "UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error") {
			t.Error("oneof_discriminator not checking child for UnmarshalJSONSebuf")
		}
	})

	// I.32 — UnmarshalJSON still present on every annotated message
	t.Run("I32_UnmarshalJSON_still_present", func(t *testing.T) {
		encodingFiles := []struct {
			file     string
			msgNames []string
		}{
			{"int64_encoding_encoding.pb.go", []string{"Int64EncodingTest"}},
			{"nullable_nullable.pb.go", []string{"User"}},
			{"timestamp_format_timestamp_format.pb.go", []string{"TimestampFormatTest"}},
			{"bytes_encoding_bytes_encoding.pb.go", []string{"BytesEncodingTest"}},
			{"flatten_flatten.pb.go", []string{"SimpleFlatten", "DualFlatten", "MixedFlatten"}},
			{"oneof_discriminator_oneof_discriminator.pb.go", []string{"FlattenedEvent", "NestedEvent"}},
		}

		for _, ef := range encodingFiles {
			content, err := os.ReadFile(filepath.Join(goldenDir, ef.file))
			if err != nil {
				t.Fatalf("Failed to read %s: %v", ef.file, err)
			}
			s := string(content)
			for _, msg := range ef.msgNames {
				wrapperSig := "func (x *" + msg + ") UnmarshalJSON(data []byte) error {"
				if !strings.Contains(s, wrapperSig) {
					t.Errorf("%s: %s missing UnmarshalJSON (json.Unmarshaler compat)", ef.file, msg)
				}
			}
		}
	})

	// I.34 — Existing call sites compile unchanged (purely additive)
	// This is tested by TestGeneratedClientCodeCompiles in golden_test.go

	// Verify enum types do NOT get UnmarshalJSONSebuf (they use direct parsing)
	t.Run("enum_types_no_UnmarshalJSONSebuf", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(goldenDir, "enum_encoding_enum_encoding.pb.go"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)

		if strings.Contains(s, "UnmarshalJSONSebuf") {
			t.Error("enum encoding should NOT have UnmarshalJSONSebuf (enums don't use protojson.Unmarshal)")
		}
		// But should still have UnmarshalJSON
		if !strings.Contains(s, "func (x *Status) UnmarshalJSON(data []byte) error {") {
			t.Error("enum encoding missing UnmarshalJSON")
		}
	})
}

// TestForwardCompatIntegration is an end-to-end integration test that generates code,
// creates a temporary Go module with httptest-based tests, and runs them.
// This covers test groups A-J from the test matrix.
//
//nolint:funlen // Integration test requires many sequential steps
func TestForwardCompatIntegration(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping integration test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-client")

	// Ensure plugin is built
	if _, statErr := os.Stat(pluginPath); os.IsNotExist(statErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	// Create temp module
	tempDir := t.TempDir()

	// Generate code for backward_compat.proto (plain proto, no custom unmarshalers)
	// This tests the core discard-unknown-fields plumbing (groups A, B, C, G, H)
	// Int64/SSE/annotation-specific tests are covered by TestForwardCompatGoldenPatterns
	genDir := filepath.Join(tempDir, "gen")
	if mkErr := os.MkdirAll(genDir, 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}

	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-client="+pluginPath,
		"--go_out="+genDir,
		"--go_opt=paths=source_relative",
		"--go-client_out="+genDir,
		"--go-client_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"backward_compat.proto",
	)
	cmd.Dir = protoDir
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Fatalf("protoc backward_compat.proto failed: %v\n%s", runErr, string(out))
	}

	// Get the sebuf module version for replace directive
	goModContent, readErr := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if readErr != nil {
		t.Fatal(readErr)
	}

	// Extract the protobuf dependency version
	var protobufVersion string
	for _, line := range strings.Split(string(goModContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "google.golang.org/protobuf") && !strings.HasPrefix(line, "module") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				protobufVersion = parts[1]
			}
		}
	}
	if protobufVersion == "" {
		t.Fatal("Could not find google.golang.org/protobuf version in go.mod")
	}

	// Write go.mod
	goMod := `module forward_compat_test

go 1.24

require (
	google.golang.org/protobuf ` + protobufVersion + `
	github.com/SebastienMelki/sebuf v0.0.0
)

replace github.com/SebastienMelki/sebuf => ` + projectRoot + `
`
	if writeErr := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Write the httptest-based test file
	testCode := `package forward_compat_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gen "forward_compat_test/gen"

	"google.golang.org/protobuf/proto"
)

// jsonHandler returns an HTTP handler that serves the given JSON body.
func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}
}

// errorHandler returns an HTTP handler that serves an error response.
func errorHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}
}

// ---- A. Default behavior (compat guarantee) ----

func TestA1_NoOption_UnknownField_PlainMessage_Rejects(t *testing.T) {
	// Server returns JSON with an unknown field for a plain proto message
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL)
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err == nil {
		t.Fatal("expected error for unknown field in strict mode, got nil")
	}
}

// ---- B. Client-level option ----

func TestB4_ClientDiscard_UnknownField_Succeeds(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err != nil {
		t.Fatalf("expected success with discard=true, got: %v", err)
	}
}

func TestB5_ClientDiscardFalse_UnknownField_Rejects(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(false))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err == nil {
		t.Fatal("expected error with explicit discard=false")
	}
}

func TestB6_ClientDefault_MatchesStrict(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	// No option set — should be strict (same as A1)
	client := gen.NewNoAnnotationsServiceClient(srv.URL)
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err == nil {
		t.Fatal("expected error with default (no option set)")
	}
}

// ---- C. Per-call option (precedence) ----

func TestC7_ClientUnset_PerCallTrue_Succeeds(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL)
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{},
		gen.WithNoAnnotationsServiceCallDiscardUnknownFields(true))
	if err != nil {
		t.Fatalf("expected success with per-call discard=true, got: %v", err)
	}
}

func TestC8_ClientTrue_PerCallUnset_Succeeds(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err != nil {
		t.Fatalf("expected success with client discard=true, got: %v", err)
	}
}

func TestC9_ClientTrue_PerCallFalse_Rejects(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{},
		gen.WithNoAnnotationsServiceCallDiscardUnknownFields(false))
	if err == nil {
		t.Fatal("expected error: per-call false should override client true")
	}
}

func TestC10_ClientFalse_PerCallTrue_Succeeds(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(false))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{},
		gen.WithNoAnnotationsServiceCallDiscardUnknownFields(true))
	if err != nil {
		t.Fatalf("expected success: per-call true should override client false, got: %v", err)
	}
}

func TestC11_ClientTrue_PerCallTrue_Succeeds(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(` + "`" + `{"unknownField": "value"}` + "`" + `))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{},
		gen.WithNoAnnotationsServiceCallDiscardUnknownFields(true))
	if err != nil {
		t.Fatalf("expected success with both true, got: %v", err)
	}
}

// ---- G. Error response path ----

func TestG28_ErrorResponse_WithDiscard_StaysStrict(t *testing.T) {
	// Server returns 400 with extras — client has discard=true but error parsing should still be strict
	errorBody := ` + "`" + `{"code": 400, "message": "bad request", "unknownField": "extra"}` + "`" + `
	srv := httptest.NewServer(errorHandler(400, errorBody))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err == nil {
		t.Fatal("expected error response")
	}
	// The error should be returned (handleErrorResponse uses false for unmarshal,
	// but falls back to raw error if strict unmarshal of ValidationError/Error fails)
	// Key: we should get an error, not a nil response
}

// ---- H. Content type branches ----

func TestH30_ProtoContentType_DiscardNoOp(t *testing.T) {
	// Server returns proto bytes — discard=true should be ignored (proto doesn't have unknowns concept at JSON level)
	msg := &gen.SimpleResponse{}
	protoBytes, _ := proto.Marshal(msg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(protoBytes)
	}))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true),
		gen.WithNoAnnotationsServiceContentType("application/x-protobuf"))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{},
		gen.WithNoAnnotationsServiceCallContentType("application/x-protobuf"))
	if err != nil {
		t.Fatalf("expected proto to work with discard=true, got: %v", err)
	}
}

func TestH31_EmptyBody_DiscardReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// empty body
	}))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err != nil {
		t.Fatalf("expected empty body to return nil with discard=true, got: %v", err)
	}
}

`

	if writeErr := os.WriteFile(filepath.Join(tempDir, "forward_compat_test.go"), []byte(testCode), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyOut, tidyErr := tidyCmd.CombinedOutput()
	if tidyErr != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", tidyErr, string(tidyOut))
	}

	// Run the tests
	testCmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	testCmd.Dir = tempDir
	testOut, testErr := testCmd.CombinedOutput()

	t.Logf("Test output:\n%s", string(testOut))

	if testErr != nil {
		t.Fatalf("integration tests failed: %v", testErr)
	}
}
