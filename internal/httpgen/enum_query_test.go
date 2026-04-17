package httpgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestEnumQueryAndPathParams exercises the generated enum handling at runtime.
// It generates code from query_params.proto, compiles it into a test binary
// with an httptest.Server, and verifies enum query + path parameter behavior.
func TestEnumQueryAndPathParams(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping enum runtime tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	// Build the plugin
	if _, statErr := os.Stat(pluginPath); os.IsNotExist(statErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	// Create temp directory for generated code + test
	tempDir := t.TempDir()
	genDir := filepath.Join(tempDir, "generated")
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		t.Fatalf("Failed to create gen dir: %v", err)
	}

	// Run protoc to generate both Go types and HTTP handlers
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+genDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+genDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"query_params.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	// Write go.mod for the temp module
	goMod := fmt.Sprintf(`module testmod

go 1.26.0

require (
	github.com/SebastienMelki/sebuf v0.0.0
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260209202127-80ab13bee0bf.1
	google.golang.org/protobuf v1.36.11
)

replace github.com/SebastienMelki/sebuf => %s
`, projectRoot)
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Write the runtime test file
	testCode := `package generated

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubServer implements QueryParamServiceServer for testing.
type stubServer struct{}

func (s *stubServer) SearchWithTypes(_ context.Context, req *SearchWithTypesRequest) (*SearchResponse, error) {
	return &SearchResponse{Total: 1}, nil
}
func (s *stubServer) SearchRequired(_ context.Context, req *SearchRequiredRequest) (*SearchResponse, error) {
	return &SearchResponse{Total: 1}, nil
}
func (s *stubServer) SearchCustomNames(_ context.Context, req *SearchCustomNamesRequest) (*SearchResponse, error) {
	return &SearchResponse{Total: 1}, nil
}
func (s *stubServer) GetWithFilters(_ context.Context, req *GetWithFiltersRequest) (*SearchResponse, error) {
	return &SearchResponse{Total: 1}, nil
}
func (s *stubServer) SearchAdvanced(_ context.Context, req *SearchAdvancedRequest) (*SearchResponse, error) {
	results := []string{req.Region.String()}
	for _, r := range req.Regions {
		results = append(results, r.String())
	}
	return &SearchResponse{
		Results: results,
		Total:   int32(req.Region),
	}, nil
}
func (s *stubServer) GetByRegion(_ context.Context, req *GetByRegionRequest) (*SearchResponse, error) {
	return &SearchResponse{
		Results: []string{req.Region.String()},
		Total:   int32(req.Region),
	}, nil
}
func (s *stubServer) GetDefaults(_ context.Context, req *EmptyRequest) (*SearchResponse, error) {
	return &SearchResponse{Total: 0}, nil
}

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	err := RegisterQueryParamServiceServer(&stubServer{}, WithMux(mux))
	if err != nil {
		t.Fatalf("Failed to register server: %v", err)
	}
	return httptest.NewServer(mux)
}

type violationResponse struct {
	Violations []struct {
		Field       string ` + "`json:\"field\"`" + `
		Description string ` + "`json:\"description\"`" + `
	} ` + "`json:\"violations\"`" + `
}

func doGet(t *testing.T, url string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

// Test 1: Query param, valid enum name
func TestEnumQuery_ValidName(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=REGION_AMERICAS")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, body)
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Total != int32(Region_REGION_AMERICAS) {
		t.Errorf("expected Total=%d (REGION_AMERICAS), got %d", Region_REGION_AMERICAS, resp.Total)
	}
}

// Test 2: Query param, valid numeric value
func TestEnumQuery_ValidNumeric(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=2")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, body)
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected Total=2 (REGION_EUROPE), got %d", resp.Total)
	}
}

// Test 3: Query param, UNSPECIFIED / 0
func TestEnumQuery_Unspecified(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	// By name
	status, _ := doGet(t, srv.URL+"/api/search/advanced?region=REGION_UNSPECIFIED")
	if status != 200 {
		t.Errorf("REGION_UNSPECIFIED by name: expected 200, got %d", status)
	}

	// By number
	status, _ = doGet(t, srv.URL+"/api/search/advanced?region=0")
	if status != 200 {
		t.Errorf("REGION_UNSPECIFIED by number (0): expected 200, got %d", status)
	}
}

// Test 4: Query param, unknown name → validation error
func TestEnumQuery_UnknownName(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=BOGUS")
	if status != 400 {
		t.Fatalf("expected 400, got %d: %s", status, body)
	}
	var vr violationResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		t.Fatalf("failed to unmarshal violations: %v", err)
	}
	if len(vr.Violations) == 0 {
		t.Fatal("expected at least one violation")
	}
	desc := vr.Violations[0].Description
	if !strings.Contains(desc, "Region") {
		t.Errorf("violation should mention enum type 'Region', got: %s", desc)
	}
}

// Test 5: Query param, unknown number → accepted (forward-compat)
func TestEnumQuery_UnknownNumber(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=99")
	if status != 200 {
		t.Fatalf("expected 200 (forward-compat), got %d: %s", status, body)
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Total != 99 {
		t.Errorf("expected Total=99 (unknown number passed through), got %d", resp.Total)
	}
}

// Test 6: Query param, wrong case → rejected (case-sensitive)
func TestEnumQuery_WrongCase(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=region_americas")
	if status != 400 {
		t.Fatalf("expected 400 (case-sensitive rejection), got %d: %s", status, body)
	}
	var vr violationResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		t.Fatalf("failed to unmarshal violations: %v", err)
	}
	if len(vr.Violations) == 0 {
		t.Fatal("expected at least one violation for wrong case")
	}
}

// Test 7: Query param, empty string → treated as unset
func TestEnumQuery_EmptyString(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?region=")
	if status != 200 {
		t.Fatalf("expected 200 (empty treated as unset), got %d: %s", status, body)
	}
}

// Test 8: Repeated enum query param (?regions=A&regions=B)
func TestEnumQuery_Repeated(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/search/advanced?regions=REGION_AMERICAS&regions=REGION_EUROPE")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, body)
	}
	var resp struct {
		Results []string ` + "`json:\"results\"`" + `
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// Results[0] is region (unset = REGION_UNSPECIFIED), Results[1] and [2] are the repeated regions
	if len(resp.Results) < 3 {
		t.Fatalf("expected at least 3 results (region + 2 repeated regions), got %d: %v", len(resp.Results), resp.Results)
	}
	if resp.Results[1] != "REGION_AMERICAS" {
		t.Errorf("expected Results[1]=REGION_AMERICAS, got %s", resp.Results[1])
	}
	if resp.Results[2] != "REGION_EUROPE" {
		t.Errorf("expected Results[2]=REGION_EUROPE, got %s", resp.Results[2])
	}
}

// Test 9a: Path param, valid enum name
func TestEnumPath_ValidName(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/regions/REGION_AMERICAS")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, body)
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Total != int32(Region_REGION_AMERICAS) {
		t.Errorf("expected Total=%d (REGION_AMERICAS), got %d", Region_REGION_AMERICAS, resp.Total)
	}
}

// Test 9b: Path param, valid number
func TestEnumPath_ValidNumber(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/regions/2")
	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, body)
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected Total=2 (REGION_EUROPE), got %d", resp.Total)
	}
}

// Test 9c: Path param, unknown name → validation error
func TestEnumPath_UnknownName(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	status, body := doGet(t, srv.URL+"/api/regions/INVALID")
	if status != 400 {
		t.Fatalf("expected 400, got %d: %s", status, body)
	}
	var vr violationResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		t.Fatalf("failed to unmarshal violations: %v", err)
	}
	if len(vr.Violations) == 0 {
		t.Fatal("expected at least one violation")
	}
	desc := vr.Violations[0].Description
	if !strings.Contains(desc, "Region") {
		t.Errorf("violation should mention enum type 'Region', got: %s", desc)
	}
}
`
	testFilePath := filepath.Join(genDir, "enum_test.go")
	if err := os.WriteFile(testFilePath, []byte(testCode), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Stderr = &stderr
	stderr.Reset()
	if tidyErr := tidyCmd.Run(); tidyErr != nil {
		t.Fatalf("go mod tidy failed: %v\nstderr: %s", tidyErr, stderr.String())
	}

	// Run the tests
	testCmd := exec.Command("go", "test", "-v", "-count=1", "./generated/")
	testCmd.Dir = tempDir
	var stdout bytes.Buffer
	testCmd.Stdout = &stdout
	stderr.Reset()
	testCmd.Stderr = &stderr

	if testErr := testCmd.Run(); testErr != nil {
		t.Fatalf("enum runtime tests failed: %v\nstdout:\n%s\nstderr:\n%s", testErr, stdout.String(), stderr.String())
	}

	t.Logf("All enum runtime tests passed:\n%s", stdout.String())
}

