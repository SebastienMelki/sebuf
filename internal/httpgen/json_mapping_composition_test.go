package httpgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestJSONMappingFeaturePairsGenerateAndBuild is the failing composition matrix
// for Task 2 of the JSON mapping composition refactor. Each pair is expected to
// generate and build once httpgen has a composed JSON-mapping emitter instead of
// per-feature method emitters.
func TestJSONMappingFeaturePairsGenerateAndBuild(t *testing.T) {
	pairs := []struct{ left, right string }{
		{"nullable", "timestamp_format"},
		{"nullable", "enum_value"},
		{"nullable", "bytes_encoding"},
		{"nullable", "empty_behavior"},
		{"nullable", "flatten"},
		{"nullable", "oneof_config"},
		{"nullable", "map_value_unwrap"},
		{"timestamp_format", "bytes_encoding"},
		{"enum_value", "bytes_encoding"},
		{"int64_number", "nullable"},
		{"int64_number", "timestamp_format"},
		{"map_value_unwrap", "timestamp_format"},
		{"root_unwrap", "bytes_encoding"},
	}

	for _, pair := range pairs {
		pair := pair
		t.Run(pair.left+"+"+pair.right, func(t *testing.T) {
			module := generateJSONMappingModule(t, jsonMappingPairProto(pair.left, pair.right), "")
			module.runGoTest(t)
		})
	}
}

// TestJSONMappingNestedDelegationRuntime covers annotated children inside an
// annotated parent. It demonstrates the desired marshal behavior before the
// composed emitter delegates child messages through MarshalJSONSebuf.
func TestJSONMappingNestedDelegationRuntime(t *testing.T) {
	module := generateJSONMappingModule(t, jsonMappingNestedDelegationProto(), jsonMappingNestedDelegationRuntimeTest())
	module.runGoTest(t)
}

// TestJSONMappingComposedMarshalRuntime covers same-message field transforms
// that must compose into a single MarshalJSONSebuf implementation.
func TestJSONMappingComposedMarshalRuntime(t *testing.T) {
	module := generateJSONMappingModule(t, jsonMappingComposedRuntimeProto(), jsonMappingComposedMarshalRuntimeTest())
	module.runGoTest(t)
}

// TestJSONMappingComposedUnmarshalRuntime covers the inverse pipeline that must
// be exposed through UnmarshalJSONSebuf for composed mappings.
func TestJSONMappingComposedUnmarshalRuntime(t *testing.T) {
	module := generateJSONMappingModule(t, jsonMappingComposedRuntimeProto(), jsonMappingComposedUnmarshalRuntimeTest())
	module.runGoTest(t)
}

type jsonMappingGeneratedModule struct {
	dir string
}

func generateJSONMappingModule(t *testing.T, protoSource, runtimeTestSource string) jsonMappingGeneratedModule {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping JSON mapping composition tests")
	}
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not found, skipping JSON mapping composition tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")

	tempDir := t.TempDir()
	protoDir := filepath.Join(tempDir, "proto")
	genDir := filepath.Join(tempDir, "gen")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("create proto dir: %v", err)
	}
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		t.Fatalf("create gen dir: %v", err)
	}

	protoPath := filepath.Join(protoDir, "composition.proto")
	if err := os.WriteFile(protoPath, []byte(protoSource), 0o644); err != nil {
		t.Fatalf("write proto fixture: %v", err)
	}

	pluginPath := filepath.Join(tempDir, "protoc-gen-go-http")
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "./cmd/protoc-gen-go-http")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build protoc-gen-go-http: %v\n%s", err, out)
	}

	protocCmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+genDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+genDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"composition.proto",
	)
	protocCmd.Dir = protoDir
	if out, err := protocCmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc JSON mapping composition fixture failed: %v\n%s", err, out)
	}

	if runtimeTestSource != "" {
		if err := os.WriteFile(filepath.Join(genDir, "json_mapping_runtime_test.go"), []byte(runtimeTestSource), 0o644); err != nil {
			t.Fatalf("write generated runtime test: %v", err)
		}
	}

	goMod := fmt.Sprintf(`module compositiontest

go 1.26.0

require (
	github.com/SebastienMelki/sebuf v0.0.0
	google.golang.org/protobuf %s
)

replace github.com/SebastienMelki/sebuf => %s
`, extractProtobufVersionFromModFile(t, projectRoot), projectRoot)
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write generated module go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy generated module: %v\n%s", err, out)
	}

	return jsonMappingGeneratedModule{dir: tempDir}
}

func (m jsonMappingGeneratedModule) runGoTest(t *testing.T) {
	t.Helper()

	cmd := exec.Command("go", "test", "./...", "-count=1", "-v")
	cmd.Dir = m.dir
	out, err := cmd.CombinedOutput()
	t.Logf("generated module test output:\n%s", out)
	if err != nil {
		t.Fatalf("generated module go test failed: %v", err)
	}
}

func jsonMappingPairProto(left, right string) string {
	features := []string{left, right}
	if left == "root_unwrap" || right == "root_unwrap" {
		other := left
		if other == "root_unwrap" {
			other = right
		}
		return jsonMappingProtoWithBody(rootUnwrapPairBody(other))
	}

	var fields bytes.Buffer
	for i, feature := range features {
		fields.WriteString(pairFeatureField(feature, i+1))
	}

	body := jsonMappingSupportDeclarations() + `
message PairSubject {
` + fields.String() + `}
`
	return jsonMappingProtoWithBody(body)
}

func pairFeatureField(feature string, number int) string {
	switch feature {
	case "int64_number":
		return fmt.Sprintf("  int64 int64_number_value = %d [(sebuf.http.int64_encoding) = INT64_ENCODING_NUMBER];\n", number)
	case "enum_value":
		return fmt.Sprintf("  PairStatus enum_value_status = %d [(sebuf.http.enum_encoding) = ENUM_ENCODING_STRING];\n", number)
	case "bytes_encoding":
		return fmt.Sprintf("  bytes bytes_encoding_value = %d [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];\n", number)
	case "timestamp_format":
		return fmt.Sprintf("  google.protobuf.Timestamp timestamp_format_value = %d [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];\n", number)
	case "nullable":
		return fmt.Sprintf("  optional string nullable_value = %d [(sebuf.http.nullable) = true];\n", number)
	case "empty_behavior":
		return fmt.Sprintf("  EmptyChild empty_behavior_value = %d [(sebuf.http.empty_behavior) = EMPTY_BEHAVIOR_NULL];\n", number)
	case "flatten":
		return fmt.Sprintf("  FlattenChild flatten_value = %d [(sebuf.http.flatten) = true, (sebuf.http.flatten_prefix) = \"flat_\"];\n", number)
	case "oneof_config":
		return fmt.Sprintf(`  oneof payload_%d {
    option (sebuf.http.oneof_config) = { discriminator: "kind" flatten: true };
    OneofText oneof_text_%d = %d [(sebuf.http.oneof_value) = "text"];
  }
`, number, number, number)
	case "map_value_unwrap":
		return fmt.Sprintf("  map<string, MapItemList> map_value_unwrap_items = %d;\n", number)
	default:
		panic("unknown JSON mapping feature: " + feature)
	}
}

func rootUnwrapPairBody(otherFeature string) string {
	var rootItemField string
	switch otherFeature {
	case "bytes_encoding":
		rootItemField = "  bytes b = 1 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];\n"
	default:
		rootItemField = "  string id = 1;\n"
	}

	return jsonMappingSupportDeclarations() + `
message RootItem {
` + rootItemField + `}

message PairSubject {
  map<string, RootItem> items = 1 [(sebuf.http.unwrap) = true];
}
`
}

func jsonMappingNestedDelegationProto() string {
	return jsonMappingProtoWithBody(`
message HexInner {
  bytes b = 1 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];
}

message NullableOuter {
  HexInner inner = 1;
  optional string n = 2 [(sebuf.http.nullable) = true];
}

message NumberInner {
  int64 id = 1 [(sebuf.http.int64_encoding) = INT64_ENCODING_NUMBER];
}

message TimestampOuter {
  NumberInner inner = 1;
  google.protobuf.Timestamp at = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
}
`)
}

func jsonMappingNestedDelegationRuntimeTest() string {
	return `package gen

import (
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNestedDelegationBytesUnderNullableParent(t *testing.T) {
	msg := &NullableOuter{Inner: &HexInner{B: []byte("Hello")}}
	got, err := msg.MarshalJSONSebuf(protojson.MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalJSONSebuf: %v", err)
	}
	want := ` + "`" + `{"inner":{"b":"48656c6c6f"},"n":null}` + "`" + `
	if string(got) != want {
		t.Fatalf("MarshalJSONSebuf = %s, want %s", got, want)
	}
}

func TestNestedDelegationInt64UnderTimestampParent(t *testing.T) {
	msg := &TimestampOuter{
		Inner: &NumberInner{Id: 12345},
		At: timestamppb.New(time.Unix(1705312200, 0)),
	}
	got, err := msg.MarshalJSONSebuf(protojson.MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalJSONSebuf: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", got, err)
	}
	inner, ok := raw["inner"].(map[string]any)
	if !ok {
		t.Fatalf("inner = %#v, want object", raw["inner"])
	}
	if gotID, ok := inner["id"].(float64); !ok || gotID != 12345 {
		t.Fatalf("inner.id = %#v, want numeric 12345", inner["id"])
	}
	if gotAt, ok := raw["at"].(float64); !ok || gotAt != 1705312200 {
		t.Fatalf("at = %#v, want numeric 1705312200", raw["at"])
	}
}
`
}

func jsonMappingComposedRuntimeProto() string {
	return jsonMappingProtoWithBody(jsonMappingSupportDeclarations() + `
message NullableTimestamp {
  optional string n = 1 [(sebuf.http.nullable) = true];
  google.protobuf.Timestamp at = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
}

message EnumNullable {
  PairStatus status = 1 [(sebuf.http.enum_encoding) = ENUM_ENCODING_STRING];
  optional string note = 2 [(sebuf.http.nullable) = true];
}

message MapUnwrapWithSibling {
  map<string, MapItemList> items = 1;
  google.protobuf.Timestamp at = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
}
`)
}

func jsonMappingComposedMarshalRuntimeTest() string {
	return `package gen

import (
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNullableAndTimestampMarshalCompose(t *testing.T) {
	msg := &NullableTimestamp{At: timestamppb.New(time.Unix(1705312200, 0))}
	got, err := msg.MarshalJSONSebuf(protojson.MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalJSONSebuf: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", got, err)
	}
	if _, ok := raw["n"]; !ok || raw["n"] != nil {
		t.Fatalf("n = %#v, want explicit null", raw["n"])
	}
	if gotAt, ok := raw["at"].(float64); !ok || gotAt != 1705312200 {
		t.Fatalf("at = %#v, want numeric 1705312200", raw["at"])
	}
}

func TestEnumValueAndNullableMarshalCompose(t *testing.T) {
	msg := &EnumNullable{Status: PairStatus_PAIR_STATUS_ACTIVE}
	got, err := msg.MarshalJSONSebuf(protojson.MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalJSONSebuf: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", got, err)
	}
	if raw["status"] != "active" {
		t.Fatalf("status = %#v, want active", raw["status"])
	}
	if _, ok := raw["note"]; !ok || raw["note"] != nil {
		t.Fatalf("note = %#v, want explicit null", raw["note"])
	}
}

func TestMapValueUnwrapAndTimestampMarshalCompose(t *testing.T) {
	msg := &MapUnwrapWithSibling{
		Items: map[string]*MapItemList{"AAPL": &MapItemList{Items: []*MapItem{{Id: "one"}}}},
		At: timestamppb.New(time.Unix(1705312200, 0)),
	}
	got, err := msg.MarshalJSONSebuf(protojson.MarshalOptions{})
	if err != nil {
		t.Fatalf("MarshalJSONSebuf: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", got, err)
	}
	if gotAt, ok := raw["at"].(float64); !ok || gotAt != 1705312200 {
		t.Fatalf("at = %#v, want numeric 1705312200", raw["at"])
	}
	items, ok := raw["items"].(map[string]any)
	if !ok {
		t.Fatalf("items = %#v, want object", raw["items"])
	}
	if _, ok := items["AAPL"].([]any); !ok {
		t.Fatalf("items.AAPL = %#v, want unwrapped array", items["AAPL"])
	}
}
`
}

func jsonMappingComposedUnmarshalRuntimeTest() string {
	return `package gen

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestNullableTimestampUnmarshalCompose(t *testing.T) {
	msg := &NullableTimestamp{}
	if err := msg.UnmarshalJSONSebuf([]byte(` + "`" + `{"n":null,"at":1705312200}` + "`" + `), protojson.UnmarshalOptions{}); err != nil {
		t.Fatalf("UnmarshalJSONSebuf: %v", err)
	}
	if msg.N != nil {
		t.Fatalf("N = %#v, want nil optional presence for explicit null", *msg.N)
	}
	if got := msg.At.AsTime().Unix(); got != 1705312200 {
		t.Fatalf("At = %d, want 1705312200", got)
	}
}

func TestEnumNullableUnmarshalCompose(t *testing.T) {
	msg := &EnumNullable{}
	if err := msg.UnmarshalJSONSebuf([]byte(` + "`" + `{"status":"active","note":null}` + "`" + `), protojson.UnmarshalOptions{}); err != nil {
		t.Fatalf("UnmarshalJSONSebuf: %v", err)
	}
	if msg.Status != PairStatus_PAIR_STATUS_ACTIVE {
		t.Fatalf("Status = %v, want PAIR_STATUS_ACTIVE", msg.Status)
	}
	if msg.Note != nil {
		t.Fatalf("Note = %#v, want nil optional presence for explicit null", *msg.Note)
	}
}

func TestMapValueUnwrapTimestampUnmarshalCompose(t *testing.T) {
	msg := &MapUnwrapWithSibling{}
	if err := msg.UnmarshalJSONSebuf([]byte(` + "`" + `{"items":{"AAPL":[{"id":"one"}]},"at":1705312200}` + "`" + `), protojson.UnmarshalOptions{}); err != nil {
		t.Fatalf("UnmarshalJSONSebuf: %v", err)
	}
	if got := msg.At.AsTime().Unix(); got != 1705312200 {
		t.Fatalf("At = %d, want 1705312200", got)
	}
	if msg.Items["AAPL"] == nil || len(msg.Items["AAPL"].Items) != 1 || msg.Items["AAPL"].Items[0].Id != "one" {
		t.Fatalf("Items[AAPL] = %#v, want one unwrapped item with id one", msg.Items["AAPL"])
	}
}
`
}

func jsonMappingSupportDeclarations() string {
	return `
enum PairStatus {
  PAIR_STATUS_UNSPECIFIED = 0;
  PAIR_STATUS_ACTIVE = 1 [(sebuf.http.enum_value) = "active"];
  PAIR_STATUS_INACTIVE = 2 [(sebuf.http.enum_value) = "inactive"];
}

message EmptyChild {
  string value = 1;
}

message FlattenChild {
  string name = 1;
}

message OneofText {
  string body = 1;
}

message MapItem {
  string id = 1;
}

message MapItemList {
  repeated MapItem items = 1 [(sebuf.http.unwrap) = true];
}
`
}

func jsonMappingProtoWithBody(body string) string {
	return `syntax = "proto3";

package composition;

import "google/protobuf/timestamp.proto";
import "sebuf/http/annotations.proto";

option go_package = "compositiontest/gen;gen";

` + body
}
