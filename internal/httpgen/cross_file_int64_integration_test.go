package httpgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCrossFileInt64EncodingIntegration is the end-to-end proof for issue #217. Golden-file
// assertions only show that a wrapper method was emitted; this test compiles the generated code
// and inspects the actual JSON bytes, which is what the issue is actually about.
//
// It:
//  1. generates Go types + encoding methods from the cross-file proto pair and the deep-nesting
//     proto (into separate packages, since their go_package options differ),
//  2. writes a temporary Go module,
//  3. asserts int64 NUMBER fields serialize as bare JSON numbers rather than quoted strings —
//     across a file boundary, at depth > 1, and on a message that has both its own NUMBER field
//     and a nested child — and that each shape round-trips.
func TestCrossFileInt64EncodingIntegration(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping integration test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	if _, statErr := os.Stat(pluginPath); os.IsNotExist(statErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	tempDir := t.TempDir()

	// Each proto set goes to its own directory: they declare different go_package values, and
	// protoc-gen-go with paths=source_relative would otherwise put two Go packages in one dir.
	generate := func(outSubdir string, protoFiles ...string) {
		outDir := filepath.Join(tempDir, outSubdir)
		if mkErr := os.MkdirAll(outDir, 0o755); mkErr != nil {
			t.Fatal(mkErr)
		}
		args := []string{
			"--plugin=protoc-gen-go-http=" + pluginPath,
			"--go_out=" + outDir,
			"--go_opt=paths=source_relative",
			"--go-http_out=" + outDir,
			"--go-http_opt=paths=source_relative",
			"--proto_path=" + protoDir,
			"--proto_path=" + filepath.Join(projectRoot, "proto"),
		}
		args = append(args, protoFiles...)

		cmd := exec.Command("protoc", args...)
		cmd.Dir = protoDir
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			t.Fatalf("protoc failed for %v: %v\n%s", protoFiles, runErr, string(out))
		}
	}

	// Both files must be compiled together: the annotated message lives in the imported file.
	generate("gen", "int64_cross_file_response.proto", "int64_cross_file_reading.proto")
	generate("deep", "int64_deep_nested_encoding.proto")

	protobufVersion := extractProtobufVersionFromModFile(t, projectRoot)
	writeCrossFileInt64TestModule(t, tempDir, projectRoot, protobufVersion)

	testCmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	testCmd.Dir = tempDir
	testOut, testErr := testCmd.CombinedOutput()

	t.Logf("Test output:\n%s", string(testOut))

	if testErr != nil {
		t.Fatalf("integration tests failed: %v", testErr)
	}
}

func writeCrossFileInt64TestModule(t *testing.T, tempDir, projectRoot, protobufVersion string) {
	t.Helper()

	goMod := `module crossfile_int64_test

go 1.24

require (
	google.golang.org/protobuf ` + protobufVersion + `
	github.com/SebastienMelki/sebuf v0.0.0
)

replace github.com/SebastienMelki/sebuf => ` + projectRoot + `
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(tempDir, "crossfile_int64_test.go"),
		[]byte(crossFileInt64IntegrationTestCode()),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyOut, tidyErr := tidyCmd.CombinedOutput()
	if tidyErr != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", tidyErr, string(tidyOut))
	}
}

// crossFileInt64IntegrationTestCode is the test source that runs inside the temp module.
// The value used (1715000000000) is the one from the issue report.
func crossFileInt64IntegrationTestCode() string {
	return `package crossfile_int64_test

import (
	"encoding/json"
	"strings"
	"testing"

	deep "crossfile_int64_test/deep"
	gen "crossfile_int64_test/gen"

	"google.golang.org/protobuf/encoding/protojson"
)

const timestampMs = 1715000000000

func TestSingularNestedInt64SerializesAsNumber(t *testing.T) {
	resp := &gen.GetSensorReadingResponse{
		Reading: &gen.SensorReading{TimestampMs: timestampMs, SensorId: "sensor-1"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(data)

	if !strings.Contains(got, ` + "`" + `"timestampMs":1715000000000` + "`" + `) {
		t.Errorf("nested int64 with int64_encoding=NUMBER did not serialize as a JSON number.\ngot: %s", got)
	}
	if strings.Contains(got, ` + "`" + `"timestampMs":"1715000000000"` + "`" + `) {
		t.Errorf("nested int64 serialized as a quoted string -- the transitive wrapper was bypassed (issue #217).\ngot: %s", got)
	}
}

func TestRepeatedNestedInt64SerializesAsNumber(t *testing.T) {
	resp := &gen.GetSensorReadingsResponse{
		Readings: []*gen.SensorReading{
			{TimestampMs: timestampMs, SensorId: "sensor-1"},
			{TimestampMs: timestampMs + 1, SensorId: "sensor-2"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(data)

	if !strings.Contains(got, ` + "`" + `"timestampMs":1715000000000` + "`" + `) ||
		!strings.Contains(got, ` + "`" + `"timestampMs":1715000000001` + "`" + `) {
		t.Errorf("repeated nested int64 did not serialize as JSON numbers.\ngot: %s", got)
	}
	if strings.Contains(got, ` + "`" + `"timestampMs":"` + "`" + `) {
		t.Errorf("repeated nested int64 serialized as quoted strings (issue #217).\ngot: %s", got)
	}
}

// Control: plain protojson still emits the quoted string. This pins down that the numeric
// output above comes from the generated wrapper and not from some protojson default.
func TestProtojsonAloneStillQuotesTheInt64(t *testing.T) {
	resp := &gen.GetSensorReadingResponse{
		Reading: &gen.SensorReading{TimestampMs: timestampMs},
	}

	data, err := protojson.Marshal(resp)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	if !strings.Contains(string(data), ` + "`" + `"1715000000000"` + "`" + `) {
		t.Fatalf("expected raw protojson to quote the int64, got: %s", string(data))
	}
}

func TestNestedInt64RoundTrips(t *testing.T) {
	t.Run("singular", func(t *testing.T) {
		body := ` + "`" + `{"reading":{"timestampMs":1715000000000,"sensorId":"sensor-1"}}` + "`" + `
		var resp gen.GetSensorReadingResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if resp.GetReading().GetTimestampMs() != timestampMs {
			t.Errorf("round-trip lost the value: got %d, want %d",
				resp.GetReading().GetTimestampMs(), timestampMs)
		}
		if resp.GetReading().GetSensorId() != "sensor-1" {
			t.Errorf("round-trip lost sibling field: got %q", resp.GetReading().GetSensorId())
		}
	})

	t.Run("repeated", func(t *testing.T) {
		body := ` + "`" + `{"readings":[{"timestampMs":1715000000000},{"timestampMs":1715000000001}]}` + "`" + `
		var resp gen.GetSensorReadingsResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if len(resp.GetReadings()) != 2 {
			t.Fatalf("expected 2 readings, got %d", len(resp.GetReadings()))
		}
		if resp.GetReadings()[0].GetTimestampMs() != timestampMs ||
			resp.GetReadings()[1].GetTimestampMs() != timestampMs+1 {
			t.Errorf("round-trip lost repeated values: %v", resp.GetReadings())
		}
	})
}

// Outer -> Middle -> Leaf: the depth > 1 chain, end to end. Middle reaches Leaf directly;
// Outer reaches it only through Middle.
func TestDeepChainInt64SerializesAsNumber(t *testing.T) {
	outer := &deep.Outer{
		Middle: &deep.Middle{Leaf: &deep.Leaf{Value: 9007199254740993}, Label: "x"},
	}

	data, err := json.Marshal(outer)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(data), ` + "`" + `"value":9007199254740993` + "`" + `) {
		t.Errorf("int64 two levels down did not serialize as a JSON number.\ngot: %s", string(data))
	}
}

// A message with BOTH a direct NUMBER field and a child that reaches one. Patching only the
// direct field leaves the child serialized by protojson, so the child's int64 stays quoted.
func TestMixedDirectAndNestedInt64SerializesAsNumber(t *testing.T) {
	root := &deep.Node{Id: 1, Child: &deep.Node{Id: 2, Child: &deep.Node{Id: 3}}}

	data, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := string(data)

	if strings.Contains(got, ` + "`" + `"id":"` + "`" + `) {
		t.Errorf("a message with direct NUMBER fields did not propagate to its nested children: "+
			"an id is still a quoted string.\ngot: %s", got)
	}
	for _, want := range []string{` + "`" + `"id":1` + "`" + `, ` + "`" + `"id":2` + "`" + `, ` + "`" + `"id":3` + "`" + `} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s at some nesting level.\ngot: %s", want, got)
		}
	}
}

func TestMixedDirectAndNestedInt64RoundTrips(t *testing.T) {
	body := ` + "`" + `{"child":{"child":{"id":3},"id":2},"id":1}` + "`" + `

	var root deep.Node
	if err := json.Unmarshal([]byte(body), &root); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if root.GetId() != 1 || root.GetChild().GetId() != 2 || root.GetChild().GetChild().GetId() != 3 {
		t.Errorf("round-trip lost values at some depth: %v", &root)
	}
}
`
}
