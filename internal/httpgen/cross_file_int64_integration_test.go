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
//  1. generates Go types + encoding methods from the cross-file proto pair,
//  2. writes a temporary Go module,
//  3. asserts timestampMs serializes as a bare JSON number (not a quoted string) for both the
//     singular and the repeated nested case, and round-trips back.
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
	genDir := filepath.Join(tempDir, "gen")
	if mkErr := os.MkdirAll(genDir, 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// Both files must be compiled together: the annotated message lives in the imported file.
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+genDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+genDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"int64_cross_file_response.proto",
		"int64_cross_file_reading.proto",
	)
	cmd.Dir = protoDir
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Fatalf("protoc failed: %v\n%s", runErr, string(out))
	}

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
	for name, body := range map[string]string{
		"singular": ` + "`" + `{"reading":{"timestampMs":1715000000000,"sensorId":"sensor-1"}}` + "`" + `,
	} {
		t.Run(name, func(t *testing.T) {
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
	}

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
`
}
