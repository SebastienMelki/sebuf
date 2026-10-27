package pyclientgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runProtocWithOptions generates internal/pyclientgen/testdata/proto/field_names.proto
// with paths=source_relative and the given further plugin options, and returns
// the directory it wrote to.
func runProtocWithOptions(t *testing.T, opt string) string {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found")
	}
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	pluginPath := filepath.Join(t.TempDir(), "protoc-gen-py-client")
	build := exec.Command("go", "build", "-o", pluginPath, "./cmd/protoc-gen-py-client")
	build.Dir = projectRoot
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build plugin: %v\n%s", buildErr, out)
	}

	outDir := t.TempDir()
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-py-client="+pluginPath,
		"--py-client_out="+outDir,
		"--py-client_opt=paths=source_relative"+opt,
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"field_names.proto",
	)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}
	return outDir
}

// pythonRoundTrip runs a Python snippet against the generated module and
// returns what it printed. The module is loaded the way the golden test does.
func pythonRoundTrip(t *testing.T, modulePath, snippet string) string {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found")
	}
	script := "import importlib.util, json, sys\n" +
		"spec = importlib.util.spec_from_file_location('m', " + pythonRepr(modulePath) + ")\n" +
		"m = importlib.util.module_from_spec(spec)\n" +
		"sys.modules['m'] = m\n" +
		"spec.loader.exec_module(m)\n" + snippet
	cmd := exec.Command("python3", "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("python failed: %v\nstderr: %s\nscript:\n%s", err, stderr.String(), script)
	}
	return strings.TrimSpace(stdout.String())
}

// A wire message with snake_case keys, as a service whose JSON keeps the proto
// field names sends it.
const snakeWire = `{"worker_name": "w1", "protocol_version": 3, "is_draining": true, ` +
	`"worker_kind": "WORKER_KIND_ML", "nested_probe": {"probe_detail": "ok", "retry_count": 2}, ` +
	`"recent_probes": [{"probe_detail": "a", "retry_count": 1}], "extra_labels": {"k": "v"}, ` +
	`"capability_set": ["x"]}`

func TestFieldNamesProto_SerializesAndParsesTheProtoFieldNames(t *testing.T) {
	outDir := runProtocWithOptions(t, ",field_names=proto")
	module := filepath.Join(outDir, "field_names_client.py")
	content, err := os.ReadFile(module)
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	for _, want := range []string{`d["worker_name"]`, `data["worker_name"]`, `d["nested_probe"]`,
		`data["recent_probes"]`, `d["extra_labels"]`, `data["next_cursor"]`, `data["probe_detail"]`} {
		if !strings.Contains(string(content), want) {
			t.Errorf("field_names=proto output missing %s", want)
		}
	}
	for _, forbidden := range []string{`"workerName"`, `"nestedProbe"`, `"recentProbes"`,
		`"extraLabels"`, `"nextCursor"`, `"probeDetail"`, `"errorCode"`} {
		if strings.Contains(string(content), forbidden) {
			t.Errorf("field_names=proto output still uses the lowerCamelCase key %s", forbidden)
		}
	}

	// from_dict reads a real snake_case wire message (before the fix it read
	// the lowerCamelCase keys, found none, and returned every default), and
	// to_dict writes the same keys back.
	got := pythonRoundTrip(t, module, `
d = json.loads('''`+snakeWire+`''')
req = m.RegisterRequest.from_dict(d)
assert req.worker_name == 'w1', req.worker_name
assert req.protocol_version == 3, req.protocol_version
assert req.is_draining is True, req.is_draining
assert req.nested_probe.probe_detail == 'ok', req.nested_probe
assert req.nested_probe.retry_count == 2
assert [p.probe_detail for p in req.recent_probes] == ['a']
assert req.extra_labels == {'k': 'v'}
assert req.capability_set == ['x']
out = req.to_dict()
assert out['worker_name'] == 'w1' and out['nested_probe']['probe_detail'] == 'ok', out
assert 'workerName' not in out
resp = m.RegisterResponse.from_dict({'next_cursor': 'c2', 'known_probes': [{'probe_detail': 'p'}]})
assert resp.next_cursor == 'c2', resp
assert resp.known_probes[0].probe_detail == 'p'
print('ok')
`)
	if got != "ok" {
		t.Fatalf("round trip printed %q", got)
	}
}

func TestFieldNamesJSON_IsTheDefault_AndStillWritesLowerCamelCase(t *testing.T) {
	for _, opt := range []string{"", ",field_names=json"} {
		outDir := runProtocWithOptions(t, opt)
		module := filepath.Join(outDir, "field_names_client.py")
		content, err := os.ReadFile(module)
		if err != nil {
			t.Fatalf("read generated client: %v", err)
		}
		for _, want := range []string{`"workerName"`, `"nestedProbe"`, `"nextCursor"`} {
			if !strings.Contains(string(content), want) {
				t.Errorf("default output (%s) missing the json_name key %s", opt, want)
			}
		}
		if strings.Contains(string(content), `d["worker_name"]`) {
			t.Errorf("default output (%s) switched to proto field names", opt)
		}
	}
}

func TestFieldNamesRejectsAnUnknownValue(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found")
	}
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	pluginPath := filepath.Join(t.TempDir(), "protoc-gen-py-client")
	build := exec.Command("go", "build", "-o", pluginPath, "./cmd/protoc-gen-py-client")
	build.Dir = projectRoot
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build plugin: %v\n%s", buildErr, out)
	}
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-py-client="+pluginPath,
		"--py-client_out="+t.TempDir(),
		"--py-client_opt=field_names=snake",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"field_names.proto",
	)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr == nil || !strings.Contains(stderr.String(), "field_names must be json or proto") {
		t.Fatalf("want a field_names validation error, got err=%v stderr=%s", runErr, stderr.String())
	}
}
