package clientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// encodingFileSpec maps a golden encoding file to the messages it should contain.
type encodingFileSpec struct {
	file     string
	msgNames []string
}

// allEncodingFiles returns the spec for all annotated message encoding golden files.
func allEncodingFiles() []encodingFileSpec {
	return []encodingFileSpec{
		{"int64_encoding_encoding.pb.go", []string{"Int64EncodingTest"}},
		{
			"int64_nested_encoding_encoding.pb.go",
			[]string{"SensorReading", "GetSensorReadingResponse", "GetMultiSensorResponse"},
		},
		{"nullable_nullable.pb.go", []string{"User"}},
		{"timestamp_format_timestamp_format.pb.go", []string{"TimestampFormatTest"}},
		{"bytes_encoding_bytes_encoding.pb.go", []string{"BytesEncodingTest"}},
		{"empty_behavior_empty_behavior.pb.go", []string{"Response"}},
		{"flatten_flatten.pb.go", []string{"SimpleFlatten", "DualFlatten", "MixedFlatten"}},
		{"oneof_discriminator_oneof_discriminator.pb.go", []string{"FlattenedEvent", "NestedEvent"}},
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatalf("Failed to read %s: %v", name, err)
	}
	return string(content)
}

// TestK36_AnnotatedMessagesHaveUnmarshalJSONSebuf verifies every annotated message
// gets the UnmarshalJSONSebuf method.
func TestK36_AnnotatedMessagesHaveUnmarshalJSONSebuf(t *testing.T) {
	for _, ef := range allEncodingFiles() {
		s := readGolden(t, ef.file)
		for _, msg := range ef.msgNames {
			pattern := "func (x *" + msg +
				") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {"
			if !strings.Contains(s, pattern) {
				t.Errorf("%s: missing UnmarshalJSONSebuf for %s", ef.file, msg)
			}
		}
	}
}

// TestK37_UnmarshalJSONDelegatesToSebuf verifies every annotated message's UnmarshalJSON
// is a thin wrapper that delegates to UnmarshalJSONSebuf.
func TestK37_UnmarshalJSONDelegatesToSebuf(t *testing.T) {
	for _, ef := range allEncodingFiles() {
		s := readGolden(t, ef.file)
		for _, msg := range ef.msgNames {
			pattern := "return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})"
			wrapperSig := "func (x *" + msg + ") UnmarshalJSON(data []byte) error {"
			if !strings.Contains(s, wrapperSig) {
				t.Errorf("%s: missing UnmarshalJSON wrapper for %s", ef.file, msg)
			}
			if !strings.Contains(s, pattern) {
				t.Errorf(
					"%s: UnmarshalJSON doesn't delegate to UnmarshalJSONSebuf for %s",
					ef.file, msg,
				)
			}
		}
	}
}

// TestSebufUnmarshalerInterfaceDefined verifies the interface is present in all client files.
func TestSebufUnmarshalerInterfaceDefined(t *testing.T) {
	clientFiles := []string{
		"backward_compat_client.pb.go",
		"int64_encoding_client.pb.go",
		"nullable_client.pb.go",
		"flatten_client.pb.go",
		"sse_client.pb.go",
	}

	for _, f := range clientFiles {
		s := readGolden(t, f)
		if !strings.Contains(s, "type sebufUnmarshaler interface {") {
			t.Errorf("%s: missing sebufUnmarshaler interface", f)
		}
		if !strings.Contains(
			s,
			"UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error",
		) {
			t.Errorf("%s: sebufUnmarshaler missing UnmarshalJSONSebuf method", f)
		}
	}
}

// TestUnmarshalResponseChecksSebufFirst verifies unmarshalResponse dispatches
// to sebufUnmarshaler before json.Unmarshaler.
func TestUnmarshalResponseChecksSebufFirst(t *testing.T) {
	clientFiles := []string{
		"backward_compat_client.pb.go",
		"int64_encoding_client.pb.go",
		"sse_client.pb.go",
	}

	for _, f := range clientFiles {
		s := readGolden(t, f)

		checks := []struct {
			pattern string
			msg     string
		}{
			{
				"unmarshalResponse(body []byte, msg proto.Message, " +
					"contentType string, discardUnknown bool) error",
				"unmarshalResponse missing discardUnknown parameter",
			},
			{
				"opts := protojson.UnmarshalOptions{DiscardUnknown: discardUnknown}",
				"unmarshalResponse not building opts from discardUnknown",
			},
			{
				"if u, ok := msg.(sebufUnmarshaler); ok {",
				"unmarshalResponse not checking sebufUnmarshaler",
			},
			{
				"return u.UnmarshalJSONSebuf(body, opts)",
				"unmarshalResponse not calling UnmarshalJSONSebuf",
			},
		}
		for _, c := range checks {
			if !strings.Contains(s, c.pattern) {
				t.Errorf("%s: %s", f, c.msg)
			}
		}
	}
}

// TestClientStructAndOptions verifies the client struct and options are generated correctly.
func TestClientStructAndOptions(t *testing.T) {
	s := readGolden(t, "backward_compat_client.pb.go")

	checks := []struct {
		pattern string
		msg     string
	}{
		{"discardUnknownFields bool", "client struct missing discardUnknownFields field"},
		{"discardUnknownFields *bool", "call options missing discardUnknownFields *bool field"},
		{
			"func WithNoAnnotationsServiceDiscardUnknownFields(discard bool)",
			"missing WithNoAnnotationsServiceDiscardUnknownFields option",
		},
		{
			"func WithNoAnnotationsServiceCallDiscardUnknownFields(discard bool)",
			"missing WithNoAnnotationsServiceCallDiscardUnknownFields option",
		},
		{"o.discardUnknownFields = &discard", "call option not using pointer assignment"},
		{"discardUnknown := c.discardUnknownFields", "RPC method not reading client-level option"},
		{
			"if callOpts.discardUnknownFields != nil {",
			"RPC method not checking per-call override",
		},
		{
			"discardUnknown = *callOpts.discardUnknownFields",
			"RPC method not applying per-call override",
		},
	}
	for _, c := range checks {
		if !strings.Contains(s, c.pattern) {
			t.Error(c.msg)
		}
	}
}

// TestG29_HandleErrorResponseStrictMode verifies error responses always use strict unmarshaling.
func TestG29_HandleErrorResponseStrictMode(t *testing.T) {
	s := readGolden(t, "backward_compat_client.pb.go")

	checks := []struct {
		pattern string
		msg     string
	}{
		{
			"c.unmarshalResponse(body, validationErr, contentType, false)",
			"handleErrorResponse not passing false (strict) for ValidationError",
		},
		{
			"c.unmarshalResponse(body, genericErr, contentType, false)",
			"handleErrorResponse not passing false (strict) for Error",
		},
		{
			"Always use strict mode (false) for error parsing",
			"handleErrorResponse missing comment explaining strict mode",
		},
	}
	for _, c := range checks {
		if !strings.Contains(s, c.pattern) {
			t.Error(c.msg)
		}
	}
}

// TestSSEEventStreamDiscardSupport verifies SSE streaming honors discardUnknownFields.
func TestSSEEventStreamDiscardSupport(t *testing.T) {
	s := readGolden(t, "sse_client.pb.go")

	checks := []struct {
		pattern string
		msg     string
	}{
		{"discardUnknownFields bool", "EventStream missing discardUnknownFields field"},
		{"discardUnknownFields: discardUnknown,", "SSE method not passing discardUnknown"},
		{
			"opts := protojson.UnmarshalOptions{DiscardUnknown: s.discardUnknownFields}",
			"EventStream.Next not building opts",
		},
		{
			"if u, ok := any(event).(sebufUnmarshaler); ok {",
			"EventStream.Next not checking sebufUnmarshaler",
		},
		{
			"unmarshalErr = u.UnmarshalJSONSebuf([]byte(data), opts)",
			"EventStream.Next not calling UnmarshalJSONSebuf",
		},
	}
	for _, c := range checks {
		if !strings.Contains(s, c.pattern) {
			t.Error(c.msg)
		}
	}
}

// TestOptsForwardingToChildren verifies wrapper, flatten, and oneof generators
// forward opts to nested children.
func TestOptsForwardingToChildren(t *testing.T) {
	files := []struct {
		file string
		name string
	}{
		{"int64_nested_encoding_encoding.pb.go", "wrapper"},
		{"flatten_flatten.pb.go", "flatten"},
		{"oneof_discriminator_oneof_discriminator.pb.go", "oneof_discriminator"},
	}

	for _, f := range files {
		s := readGolden(t, f.file)
		pattern := "UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error"
		if !strings.Contains(s, pattern) {
			t.Errorf("%s not checking child for UnmarshalJSONSebuf", f.name)
		}
	}
}

// TestI32_UnmarshalJSONStillPresent verifies backward compat with json.Unmarshaler.
func TestI32_UnmarshalJSONStillPresent(t *testing.T) {
	encodingFiles := []encodingFileSpec{
		{"int64_encoding_encoding.pb.go", []string{"Int64EncodingTest"}},
		{"nullable_nullable.pb.go", []string{"User"}},
		{"timestamp_format_timestamp_format.pb.go", []string{"TimestampFormatTest"}},
		{"bytes_encoding_bytes_encoding.pb.go", []string{"BytesEncodingTest"}},
		{"flatten_flatten.pb.go", []string{"SimpleFlatten", "DualFlatten", "MixedFlatten"}},
		{"oneof_discriminator_oneof_discriminator.pb.go", []string{"FlattenedEvent", "NestedEvent"}},
	}

	for _, ef := range encodingFiles {
		s := readGolden(t, ef.file)
		for _, msg := range ef.msgNames {
			wrapperSig := "func (x *" + msg + ") UnmarshalJSON(data []byte) error {"
			if !strings.Contains(s, wrapperSig) {
				t.Errorf(
					"%s: %s missing UnmarshalJSON (json.Unmarshaler compat)",
					ef.file, msg,
				)
			}
		}
	}
}

// TestRepeatedQueryParamsUseAddNotSet verifies that repeated fields (including
// repeated enums) use for-range + queryParams.Add() instead of scalar zero-value
// checks + queryParams.Set(). Regression test for #186.
func TestRepeatedQueryParamsUseAddNotSet(t *testing.T) {
	s := readGolden(t, "query_params_client.pb.go")

	// Repeated fields must use for-range with Add()
	repeatedFields := []struct {
		fieldName string
		paramName string
	}{
		{"Countries", "countries"},
		{"Years", "years"},
		{"Flags", "flags"},
		{"Regions", "regions"},
	}

	for _, rf := range repeatedFields {
		// Must contain for-range loop with Add()
		addPattern := "queryParams.Add(\"" + rf.paramName + "\", fmt.Sprint(v))"
		if !strings.Contains(s, addPattern) {
			t.Errorf("repeated field %s: missing queryParams.Add() pattern", rf.fieldName)
		}

		rangePattern := "for _, v := range req." + rf.fieldName + " {"
		if !strings.Contains(s, rangePattern) {
			t.Errorf("repeated field %s: missing for-range loop", rf.fieldName)
		}

		// Must NOT contain scalar zero-value check with Set()
		setPattern := "queryParams.Set(\"" + rf.paramName + "\""
		if strings.Contains(s, setPattern) {
			t.Errorf("repeated field %s: should use Add() not Set()", rf.fieldName)
		}
	}

	// Scalar fields must still use Set() (not Add())
	scalarChecks := []struct {
		fieldName string
		paramName string
		zeroCheck string
	}{
		{"Region", "region", "req.Region != 0"},
		{"Keyword", "keyword", `req.Keyword != ""`},
	}

	for _, sc := range scalarChecks {
		setPattern := "queryParams.Set(\"" + sc.paramName + "\""
		if !strings.Contains(s, setPattern) {
			t.Errorf("scalar field %s: missing queryParams.Set()", sc.fieldName)
		}

		if !strings.Contains(s, sc.zeroCheck) {
			t.Errorf("scalar field %s: missing zero-value check %q", sc.fieldName, sc.zeroCheck)
		}
	}
}

// TestEnumTypesNoUnmarshalJSONSebuf verifies enums don't get the new interface.
func TestEnumTypesNoUnmarshalJSONSebuf(t *testing.T) {
	s := readGolden(t, "enum_encoding_enum_encoding.pb.go")

	if strings.Contains(s, "UnmarshalJSONSebuf") {
		t.Error("enum encoding should NOT have UnmarshalJSONSebuf")
	}
	if !strings.Contains(s, "func (x *Status) UnmarshalJSON(data []byte) error {") {
		t.Error("enum encoding missing UnmarshalJSON")
	}
}

// TestForwardCompatIntegration is an end-to-end integration test that generates code,
// creates a temporary Go module with httptest-based tests, and runs them.
// This covers test groups A-J from the test matrix.
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

	tempDir := t.TempDir()
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
		t.Fatalf("protoc failed: %v\n%s", runErr, string(out))
	}

	protobufVersion := extractProtobufVersion(t, projectRoot)
	writeTestModule(t, tempDir, projectRoot, protobufVersion)

	// Run the tests
	testCmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	testCmd.Dir = tempDir
	testOut, testErr := testCmd.CombinedOutput()

	t.Logf("Test output:\n%s", string(testOut))

	if testErr != nil {
		t.Fatalf("integration tests failed: %v", testErr)
	}
}

func extractProtobufVersion(t *testing.T, projectRoot string) string {
	t.Helper()
	goModContent, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range strings.Split(string(goModContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "google.golang.org/protobuf") &&
			!strings.HasPrefix(line, "module") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	t.Fatal("Could not find google.golang.org/protobuf version in go.mod")
	return ""
}

func writeTestModule(
	t *testing.T,
	tempDir, projectRoot, protobufVersion string,
) {
	t.Helper()

	goMod := `module forward_compat_test

go 1.24

require (
	google.golang.org/protobuf ` + protobufVersion + `
	github.com/SebastienMelki/sebuf v0.0.0
)

replace github.com/SebastienMelki/sebuf => ` + projectRoot + `
`
	if err := os.WriteFile(
		filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	testCode := generateIntegrationTestCode()
	if err := os.WriteFile(
		filepath.Join(tempDir, "forward_compat_test.go"), []byte(testCode), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyOut, tidyErr := tidyCmd.CombinedOutput()
	if tidyErr != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", tidyErr, string(tidyOut))
	}
}

func generateIntegrationTestCode() string {
	return `package forward_compat_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gen "forward_compat_test/gen"

	"google.golang.org/protobuf/proto"
)

func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}
}

func errorHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}
}

// ---- A. Default behavior ----

func TestA1_NoOption_UnknownField_PlainMessage_Rejects(t *testing.T) {
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
		t.Fatalf("per-call true should override client false, got: %v", err)
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
	errorBody := ` + "`" + `{"code": 400, "message": "bad request", "unknownField": "extra"}` + "`" + `
	srv := httptest.NewServer(errorHandler(400, errorBody))
	defer srv.Close()

	client := gen.NewNoAnnotationsServiceClient(srv.URL,
		gen.WithNoAnnotationsServiceDiscardUnknownFields(true))
	_, err := client.SimpleAction(context.Background(), &gen.SimpleRequest{})
	if err == nil {
		t.Fatal("expected error response")
	}
}

// ---- H. Content type branches ----

func TestH30_ProtoContentType_DiscardNoOp(t *testing.T) {
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
}
