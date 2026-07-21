package tsservergen

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
)

// TestTSServerGenGoldenFiles tests TypeScript server generation against golden files.
// Each fixture is generated into its own temp directory; every emitted .ts file
// (type modules, the slimmed server module, and errors.ts) is compared against
// testdata/golden/<relative path>.
//
// To update golden files after intentional changes:
//
//	UPDATE_GOLDEN=1 go test -run TestTSServerGenGoldenFiles
func TestTSServerGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	testCases := []struct {
		name       string
		protoFiles []string
		// assertImportFile/assertImport, when set, require the generated file at
		// assertImportFile (relative to the output dir) to contain assertImport.
		// Used to lock in cross-package relative imports in the modules layout.
		assertImportFile string
		assertImport     string
		// assertBarrelFile/assertBarrelContains, when set, require the generated
		// per-package barrel at assertBarrelFile to contain every listed
		// substring. Used to lock in that a package barrel re-exports both its
		// type module and its server module.
		assertBarrelFile     string
		assertBarrelContains []string
	}{
		{name: "comprehensive HTTP verbs", protoFiles: []string{"http_verbs_comprehensive.proto"}},
		{name: "query parameters", protoFiles: []string{"query_params.proto"}},
		{name: "backward compatibility", protoFiles: []string{"backward_compat.proto"}},
		{name: "complex features", protoFiles: []string{"complex_features.proto"}},
		{name: "unwrap variants", protoFiles: []string{"unwrap.proto"}},
		{name: "int64 encoding", protoFiles: []string{"int64_encoding.proto"}},
		{name: "enum encoding", protoFiles: []string{"enum_encoding.proto"}},
		{name: "nullable fields", protoFiles: []string{"nullable.proto"}},
		{name: "empty behavior", protoFiles: []string{"empty_behavior.proto"}},
		{name: "timestamp format", protoFiles: []string{"timestamp_format.proto"}},
		{name: "bytes encoding", protoFiles: []string{"bytes_encoding.proto"}},
		{name: "flatten", protoFiles: []string{"flatten.proto"}},
		{name: "oneof discriminator", protoFiles: []string{"oneof_discriminator.proto"}},
		{name: "multi-word oneof name", protoFiles: []string{"multi_word_oneof.proto"}},
		{name: "two un-annotated oneofs in one message", protoFiles: []string{"two_oneofs.proto"}},
		{name: "SSE streaming", protoFiles: []string{"sse.proto"}},
		{name: "empty request body", protoFiles: []string{"empty_request_body.proto"}},
		{name: "record map collision", protoFiles: []string{"record_map_collision.proto"}},
		{
			name:             "reserved error-helper names",
			protoFiles:       []string{"reserved_name.proto"},
			assertImportFile: "reserved_name_server.ts",
			assertImport:     `ValidationError as ValidationError_1`,
		},
		{
			name:             "cross-package imports",
			protoFiles:       []string{"crosspkg/common/v1/types.proto", "crosspkg/shop/v1/service.proto"},
			assertImportFile: filepath.Join("crosspkg", "shop", "v1", "service.ts"),
			assertImport:     `from "../../common/v1/types.js"`,
			assertBarrelFile: filepath.Join("crosspkg", "shop", "v1", "index.ts"),
			assertBarrelContains: []string{
				`export * from "./service.js";`,
				`export * from "./service_server.js";`,
			},
		},
		{
			name: "nested type name collision",
			protoFiles: []string{
				"nestedcollision/v1/nested_collision.proto",
				"nestedcollision/v1/wrapper.proto",
			},
			assertImportFile: filepath.Join("nestedcollision", "v1", "nested_collision.ts"),
			assertImport:     `from "./wrapper.js"`,
			assertBarrelFile: filepath.Join("nestedcollision", "v1", "index.ts"),
			assertBarrelContains: []string{
				`export * from "./nested_collision.js";`,
				`export * from "./nested_collision_server.js";`,
				`export * from "./wrapper.js";`,
			},
		},
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")

	if mkdirErr := os.MkdirAll(goldenDir, 0o755); mkdirErr != nil {
		t.Fatalf("Failed to create golden directory: %v", mkdirErr)
	}

	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-server")

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, pf := range tc.protoFiles {
				if _, statErr := os.Stat(filepath.Join(protoDir, pf)); os.IsNotExist(statErr) {
					t.Fatalf("Proto file not found: %s", pf)
				}
			}

			outDir := t.TempDir()
			args := []string{
				"--plugin=protoc-gen-ts-server=" + pluginPath,
				"--ts-server_out=" + outDir,
				"--ts-server_opt=paths=source_relative",
				"--proto_path=" + protoDir,
				"--proto_path=" + filepath.Join(projectRoot, "proto"),
			}
			args = append(args, tc.protoFiles...)
			cmd := exec.Command("protoc", args...)
			cmd.Dir = protoDir
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if runErr := cmd.Run(); runErr != nil {
				t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
			}

			if tc.assertImport != "" {
				emitted, readErr := os.ReadFile(filepath.Join(outDir, tc.assertImportFile))
				if readErr != nil {
					t.Fatalf("Failed to read generated file %s for import assertion: %v", tc.assertImportFile, readErr)
				}
				if !strings.Contains(string(emitted), tc.assertImport) {
					t.Errorf("generated %s does not contain expected cross-package import %q\n---\n%s",
						tc.assertImportFile, tc.assertImport, string(emitted))
				}
			}

			if tc.assertBarrelFile != "" {
				barrel, readErr := os.ReadFile(filepath.Join(outDir, tc.assertBarrelFile))
				if readErr != nil {
					t.Fatalf("Failed to read generated barrel %s for assertion: %v", tc.assertBarrelFile, readErr)
				}
				for _, want := range tc.assertBarrelContains {
					if !strings.Contains(string(barrel), want) {
						t.Errorf("generated barrel %s does not contain expected re-export %q\n---\n%s",
							tc.assertBarrelFile, want, string(barrel))
					}
				}
			}

			for _, rel := range generatedTSFiles(t, outDir) {
				generatedContent, readErr := os.ReadFile(filepath.Join(outDir, rel))
				if readErr != nil {
					t.Fatalf("Failed to read generated file %s: %v", rel, readErr)
				}
				goldenPath := filepath.Join(goldenDir, rel)
				if updateGolden {
					updateGoldenFile(t, goldenPath, generatedContent)
					continue
				}
				compareGoldenFile(t, rel, goldenPath, generatedContent)
			}
		})
	}
}

// generatedTSFiles returns the relative paths of every .ts file under dir, sorted.
func generatedTSFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".ts") {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, rel)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("Failed to walk generated dir: %v", walkErr)
	}
	sort.Strings(files)
	return files
}

func updateGoldenFile(t *testing.T, goldenPath string, content []byte) {
	t.Helper()
	if mkErr := os.MkdirAll(filepath.Dir(goldenPath), 0o755); mkErr != nil {
		t.Fatalf("Failed to create golden dir for %s: %v", goldenPath, mkErr)
	}
	if writeErr := os.WriteFile(goldenPath, content, 0o644); writeErr != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, writeErr)
	}
	t.Logf("Updated golden file: %s", goldenPath)
}

func compareGoldenFile(t *testing.T, name, goldenPath string, generatedContent []byte) {
	t.Helper()
	goldenContent, goldenReadErr := os.ReadFile(goldenPath)
	if goldenReadErr != nil {
		if os.IsNotExist(goldenReadErr) {
			t.Fatalf("Golden file not found: %s\nRun with UPDATE_GOLDEN=1 to create it", goldenPath)
		}
		t.Fatalf("Failed to read golden file %s: %v", goldenPath, goldenReadErr)
	}

	if !bytes.Equal(generatedContent, goldenContent) {
		t.Errorf("Generated file %s does not match golden file.\n"+
			"Run with UPDATE_GOLDEN=1 to update golden files after reviewing changes.\n"+
			"Diff:\n%s",
			name,
			diffStrings(string(goldenContent), string(generatedContent)))
	}
}

// TestTSServerGenValidationErrors verifies that the generator fails with clear errors
// for invalid proto definitions (e.g., path params without matching fields, unreachable fields).
func TestTSServerGenValidationErrors(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping validation error tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-server")

	testCases := []struct {
		name      string
		protoFile string
		wantErr   string // substring expected in stderr
	}{
		{
			name:      "path param without matching field",
			protoFile: "invalid_path_param.proto",
			wantErr:   "path parameter {id} has no matching field on request message GetItemRequest",
		},
		{
			name:      "unreachable field on GET method",
			protoFile: "invalid_uncovered_field.proto",
			wantErr:   "fields [category] on request message GetItemRequest are not reachable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-ts-server="+pluginPath,
				"--ts-server_out="+tempDir,
				"--ts-server_opt=paths=source_relative",
				"--proto_path="+protoDir,
				"--proto_path="+filepath.Join(projectRoot, "proto"),
				tc.protoFile,
			)
			cmd.Dir = protoDir

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			runErr := cmd.Run()
			if runErr == nil {
				t.Fatalf("expected protoc to fail for %s, but it succeeded", tc.protoFile)
			}

			stderrStr := stderr.String()
			if !strings.Contains(stderrStr, tc.wantErr) {
				t.Errorf("expected stderr to contain %q, got:\n%s", tc.wantErr, stderrStr)
			}
		})
	}
}

func diffStrings(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder
	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	diffCount := 0
	const maxDiffs = 20

	for i := 0; i < maxLines && diffCount < maxDiffs; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			diff.WriteString("Line ")
			diff.WriteRune(rune('0' + i/100))
			diff.WriteRune(rune('0' + (i/10)%10))
			diff.WriteRune(rune('0' + i%10))
			diff.WriteString(":\n")
			diff.WriteString("  expected: ")
			diff.WriteString(expLine)
			diff.WriteString("\n  actual:   ")
			diff.WriteString(actLine)
			diff.WriteString("\n")
			diffCount++
		}
	}

	if diffCount >= maxDiffs {
		diff.WriteString("... (more differences truncated)\n")
	}

	return diff.String()
}
