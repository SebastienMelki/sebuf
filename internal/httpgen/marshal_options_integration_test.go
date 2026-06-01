package httpgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMarshalOptionsIntegration is an end-to-end integration test that:
//  1. generates Go HTTP server code from backward_compat.proto using the freshly built plugin,
//  2. writes a temporary Go module that wires up an httptest server,
//  3. verifies the WithMarshalOptions ServerOption controls protojson output.
//
// The proto defines ActionResponse with a bool field `success`. With default
// MarshalOptions{}, false is omitted (proto3 default behavior); with
// EmitUnpopulated: true it must be serialized as `"success": false`. This is the
// exact knob the downstream contracts/ipo NoNewOrders use case needs.
func TestMarshalOptionsIntegration(t *testing.T) {
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

	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+genDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+genDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"backward_compat.proto",
	)
	cmd.Dir = protoDir
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Fatalf("protoc failed: %v\n%s", runErr, string(out))
	}

	protobufVersion := extractProtobufVersionFromModFile(t, projectRoot)
	writeMarshalOptionsTestModule(t, tempDir, projectRoot, protobufVersion)

	testCmd := exec.Command("go", "test", "-v", "-count=1", "./...")
	testCmd.Dir = tempDir
	testOut, testErr := testCmd.CombinedOutput()

	t.Logf("Test output:\n%s", string(testOut))

	if testErr != nil {
		t.Fatalf("integration tests failed: %v", testErr)
	}
}

func extractProtobufVersionFromModFile(t *testing.T, projectRoot string) string {
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

func writeMarshalOptionsTestModule(
	t *testing.T,
	tempDir, projectRoot, protobufVersion string,
) {
	t.Helper()

	goMod := `module marshal_opts_test

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

	if err := os.WriteFile(
		filepath.Join(tempDir, "marshal_opts_test.go"),
		[]byte(marshalOptionsIntegrationTestCode()),
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

// marshalOptionsIntegrationTestCode is the test source that runs inside the temp module.
// It registers the generated HTTP handler against a real ServeMux, hits it via httptest,
// and inspects the JSON body to verify EmitUnpopulated propagated through the marshal
// pipeline.
func marshalOptionsIntegrationTestCode() string {
	return `package marshal_opts_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gen "marshal_opts_test/gen"

	"google.golang.org/protobuf/encoding/protojson"
)

// stubServer implements BasePathOnlyService and returns a response whose bool
// field is false (proto3 zero value). With default protojson marshal, the field
// is omitted; with EmitUnpopulated, it must appear in the body.
type stubServer struct{}

func (stubServer) ActionOne(_ context.Context, _ *gen.ActionRequest) (*gen.ActionResponse, error) {
	return &gen.ActionResponse{Success: false}, nil
}

func (stubServer) ActionTwo(_ context.Context, _ *gen.ActionRequest) (*gen.ActionResponse, error) {
	return &gen.ActionResponse{Success: false}, nil
}

func hitActionOne(t *testing.T, opts ...gen.ServerOption) []byte {
	t.Helper()
	mux := http.NewServeMux()
	allOpts := append([]gen.ServerOption{gen.WithMux(mux)}, opts...)
	if err := gen.RegisterBasePathOnlyServiceServer(stubServer{}, allOpts...); err != nil {
		t.Fatalf("RegisterBasePathOnlyServiceServer: %v", err)
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	body := bytes.NewReader([]byte(` + "`" + `{"name":"x"}` + "`" + `))
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v2/action_one", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return respBody
}

// TestMarshalOptions_DefaultOmitsZeroBool documents the pre-existing behavior:
// without WithMarshalOptions, a bool false is omitted from the JSON body (proto3
// default protojson behavior). This is the back-compat guarantee.
func TestMarshalOptions_DefaultOmitsZeroBool(t *testing.T) {
	body := hitActionOne(t)
	if strings.Contains(string(body), "success") {
		t.Errorf("default config should omit zero-value bool, got body: %s", body)
	}
}

// TestMarshalOptions_EmitUnpopulatedEmitsZeroBool is the headline test for #178:
// WithMarshalOptions(EmitUnpopulated: true) must surface bool false in the body
// — the exact knob the contracts/ipo NoNewOrders use case requires.
func TestMarshalOptions_EmitUnpopulatedEmitsZeroBool(t *testing.T) {
	body := hitActionOne(t, gen.WithMarshalOptions(protojson.MarshalOptions{EmitUnpopulated: true}))
	got := string(body)
	if !strings.Contains(got, ` + "`" + `"success":false` + "`" + `) {
		t.Errorf("EmitUnpopulated should emit success:false, got body: %s", got)
	}
}

// TestMarshalOptions_UseProtoNamesRoundtrip verifies non-default MarshalOptions
// beyond EmitUnpopulated also flow through: with UseProtoNames=true the field
// should use the proto snake_case name rather than camelCase. ActionResponse has
// a single-word field so this is mainly a smoke test that the option isn't
// dropped on the way through marshalJSONWithOpts.
func TestMarshalOptions_UseProtoNamesRoundtrip(t *testing.T) {
	body := hitActionOne(t, gen.WithMarshalOptions(protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}))
	got := string(body)
	if !strings.Contains(got, ` + "`" + `"success":false` + "`" + `) {
		t.Errorf("combined options should still emit success:false, got body: %s", got)
	}
}
`
}
