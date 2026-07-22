package tsclientgen

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

// TestTSClientGenGoldenFiles tests TypeScript client generation against golden files.
// Each fixture is generated into its own temp directory; every emitted .ts file
// (type modules, the slimmed client module, and errors.ts) is compared against
// testdata/golden/<relative path>.
//
// To update golden files after intentional changes:
//
//	UPDATE_GOLDEN=1 go test -run TestTSClientGenGoldenFiles
func TestTSClientGenGoldenFiles(t *testing.T) {
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
		// type module and its client module.
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
		{name: "un-annotated oneof with enum and timestamp variants", protoFiles: []string{"oneof_field_typing.proto"}},
		{name: "flatten oneof unset arm guards child keys", protoFiles: []string{"flatten_oneof_unset.proto"}},
		{name: "SSE streaming", protoFiles: []string{"sse.proto"}},
		{name: "empty request body", protoFiles: []string{"empty_request_body.proto"}},
		{name: "record map collision", protoFiles: []string{"record_map_collision.proto"}},
		{
			name:             "reserved error-helper names",
			protoFiles:       []string{"reserved_name.proto"},
			assertImportFile: "reserved_name_client.ts",
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
				`export * from "./service_client.js";`,
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
				`export * from "./nested_collision_client.js";`,
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

	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-client")

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
				"--plugin=protoc-gen-ts-client=" + pluginPath,
				"--ts-client_out=" + outDir,
				"--ts-client_opt=paths=source_relative",
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

// TestTSClientGenESGoldenFiles tests protobuf-es transport client generation
// (ts_runtime=protobuf-es) against golden files. It runs protoc twice into one
// output dir: once with protoc-gen-es (emitting <proto>_pb.ts message schemas)
// and once with the sebuf ts-client plugin in es mode (emitting the transport
// client that routes every request/response through create/toJson/fromJson).
// Every emitted .ts file is compared against testdata/golden/es/.
//
// To update golden files after intentional changes (protoc-gen-es must be on PATH):
//
//	UPDATE_GOLDEN=1 go test -run TestTSClientGenESGoldenFiles
func TestTSClientGenESGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}
	esPluginPath, esErr := exec.LookPath("protoc-gen-es")
	if esErr != nil {
		t.Skip("protoc-gen-es not found, skipping protobuf-es golden file tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden", "es")

	if mkdirErr := os.MkdirAll(goldenDir, 0o755); mkdirErr != nil {
		t.Fatalf("Failed to create golden directory: %v", mkdirErr)
	}

	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-client")

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	testCases := []struct {
		name      string
		protoFile string
	}{
		// Smallest fixture exercising request-body encode + response decode.
		{name: "unary multi-word oneof", protoFile: "multi_word_oneof.proto"},
		// Server-streaming (SSE) fixture: async generators decode each event
		// through fromJson. Mirrors the hand-rolled SSE_streaming case.
		{name: "SSE streaming", protoFile: "sse.proto"},
		// Unary GET with a string path param and scalar (non-enum) query params.
		{name: "unary GET path+query params", protoFile: "get_params.proto"},
		// Service with typed headers: its RequestOptions must extend the shared
		// RequestOptions base and add only the header properties.
		{name: "typed headers", protoFile: "es_headers.proto"},
	}

	protoPaths := []string{"--proto_path=" + protoDir, "--proto_path=" + filepath.Join(projectRoot, "proto")}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, statErr := os.Stat(filepath.Join(protoDir, tc.protoFile)); os.IsNotExist(statErr) {
				t.Fatalf("Proto file not found: %s", tc.protoFile)
			}

			outDir := t.TempDir()

			// Pass 1: protoc-gen-es emits <proto>_pb.ts (imported by the client).
			esArgs := []string{
				"--plugin=protoc-gen-es=" + esPluginPath,
				"--es_out=" + outDir,
				"--es_opt=target=ts,import_extension=js",
			}
			esArgs = append(esArgs, protoPaths...)
			esArgs = append(esArgs, tc.protoFile)
			esCmd := exec.Command("protoc", esArgs...)
			esCmd.Dir = protoDir
			var esStderr bytes.Buffer
			esCmd.Stderr = &esStderr
			if runErr := esCmd.Run(); runErr != nil {
				t.Fatalf("protoc (protoc-gen-es) failed: %v\nstderr: %s", runErr, esStderr.String())
			}

			// Pass 2: sebuf ts-client in protobuf-es mode emits the transport client.
			clientArgs := []string{
				"--plugin=protoc-gen-ts-client=" + pluginPath,
				"--ts-client_out=" + outDir,
				"--ts-client_opt=paths=source_relative,ts_runtime=protobuf-es",
			}
			clientArgs = append(clientArgs, protoPaths...)
			clientArgs = append(clientArgs, tc.protoFile)
			clientCmd := exec.Command("protoc", clientArgs...)
			clientCmd.Dir = protoDir
			var clientStderr bytes.Buffer
			clientCmd.Stderr = &clientStderr
			if runErr := clientCmd.Run(); runErr != nil {
				t.Fatalf("protoc (ts-client) failed: %v\nstderr: %s", runErr, clientStderr.String())
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

// TestTSClientGenESResultGoldenFiles drives protoc-gen-es + the sebuf ts-client
// in protobuf-es mode WITH ts_error_handling=result, comparing every emitted .ts
// against testdata/golden/es-result/. This locks down the Result return shape,
// the shared result.ts (Result type, ClientError union, decodeError), and the
// error registry — including the empty-registry (no proto *Error) case.
//
// To update golden files (protoc-gen-es must be on PATH):
//
//	UPDATE_GOLDEN=1 go test -run TestTSClientGenESResultGoldenFiles
func TestTSClientGenESResultGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}
	esPluginPath, esErr := exec.LookPath("protoc-gen-es")
	if esErr != nil {
		t.Skip("protoc-gen-es not found, skipping protobuf-es golden file tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-ts-client")
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	// Each fixture gets its own golden subdirectory: result.ts is
	// invocation-global (its ClientError union + registry depend on the whole
	// proto set), so distinct single-proto invocations would otherwise clobber a
	// shared result.ts. The typecheck test compiles the whole es-result tree.
	testCases := []struct {
		name      string
		dir       string
		protoFile string
	}{
		// Service with proto *Error messages: exercises the typed ClientError
		// union + non-empty structural registry.
		{name: "typed errors", dir: "typed-errors", protoFile: "result_errors.proto"},
		// Service with NO proto *Error: ClientError is ValidationError | ApiError
		// and the registry is empty (still must compile).
		{name: "no proto errors", dir: "no-errors", protoFile: "get_params.proto"},
		// Mixed unary + SSE: the unary method returns a Result; the SSE method
		// keeps its AsyncGenerator shape and still throws via handleError (which
		// must therefore still be emitted).
		{name: "unary + SSE", dir: "sse", protoFile: "sse.proto"},
	}

	protoPaths := []string{"--proto_path=" + protoDir, "--proto_path=" + filepath.Join(projectRoot, "proto")}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, statErr := os.Stat(filepath.Join(protoDir, tc.protoFile)); os.IsNotExist(statErr) {
				t.Fatalf("Proto file not found: %s", tc.protoFile)
			}

			goldenDir := filepath.Join(baseDir, "testdata", "golden", "es-result", tc.dir)
			if mkdirErr := os.MkdirAll(goldenDir, 0o755); mkdirErr != nil {
				t.Fatalf("Failed to create golden directory: %v", mkdirErr)
			}

			outDir := t.TempDir()

			// Pass 1: protoc-gen-es emits <proto>_pb.ts (imported by the client).
			esArgs := []string{
				"--plugin=protoc-gen-es=" + esPluginPath,
				"--es_out=" + outDir,
				"--es_opt=target=ts,import_extension=js",
			}
			esArgs = append(esArgs, protoPaths...)
			esArgs = append(esArgs, tc.protoFile)
			esCmd := exec.Command("protoc", esArgs...)
			esCmd.Dir = protoDir
			var esStderr bytes.Buffer
			esCmd.Stderr = &esStderr
			if runErr := esCmd.Run(); runErr != nil {
				t.Fatalf("protoc (protoc-gen-es) failed: %v\nstderr: %s", runErr, esStderr.String())
			}

			// Pass 2: sebuf ts-client in es + Result mode.
			clientArgs := []string{
				"--plugin=protoc-gen-ts-client=" + pluginPath,
				"--ts-client_out=" + outDir,
				"--ts-client_opt=paths=source_relative,ts_runtime=protobuf-es,ts_error_handling=result",
			}
			clientArgs = append(clientArgs, protoPaths...)
			clientArgs = append(clientArgs, tc.protoFile)
			clientCmd := exec.Command("protoc", clientArgs...)
			clientCmd.Dir = protoDir
			var clientStderr bytes.Buffer
			clientCmd.Stderr = &clientStderr
			if runErr := clientCmd.Run(); runErr != nil {
				t.Fatalf("protoc (ts-client) failed: %v\nstderr: %s", runErr, clientStderr.String())
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

// readGoldenConcat reads and concatenates the named golden files. In the modules
// layout a message's interfaces and oneof union types live in the per-proto type
// module while the client module imports them, so oneof-shape assertions read both.
func readGoldenConcat(t *testing.T, names ...string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	var sb strings.Builder
	for _, name := range names {
		goldenPath := filepath.Join(wd, "testdata", "golden", name)
		content, readErr := os.ReadFile(goldenPath)
		if readErr != nil {
			t.Fatalf("Failed to read golden file %s: %v", goldenPath, readErr)
		}
		sb.Write(content)
	}
	return sb.String()
}

// TestMultiWordOneofNameDoesNotLeak asserts a multi-word oneof name
// (super_title_image) surfaces only as the PascalCase union type name and never
// leaks into the generated TypeScript as a raw snake_case wrapper property. In
// the modules layout the union type lives in the type module and the client
// imports it, so both emitted files are checked. See internal/tscommon/types.go
// (GenerateOneofDiscriminatedUnionTypeCtx / GenerateStandardInterfaceCtx).
func TestMultiWordOneofNameDoesNotLeak(t *testing.T) {
	ts := readGoldenConcat(t, "multi_word_oneof.ts", "multi_word_oneof_client.ts")

	// The oneof name renders as the PascalCase discriminated-union type name.
	if !strings.Contains(ts, "MultiWordEventSuperTitleImage") {
		t.Error("expected generated TS to contain the PascalCase union type MultiWordEventSuperTitleImage")
	}

	// The raw snake_case oneof name must never appear: no wrapper property such
	// as `super_title_image?:` leaks onto the message interface.
	if strings.Contains(ts, "super_title_image") {
		t.Error("generated TS must not contain the raw snake_case oneof name super_title_image")
	}
}

// TestTwoOneofsRenderAsIndependentPresenceUnions asserts that a message with two
// distinct un-annotated oneofs renders as an intersection of a base interface and
// one presence-discriminated union per oneof, and that each union's `?: never`
// presence guards cover only its own siblings — the two oneofs are independent, so
// neither union references the other's variant keys. In the modules layout these
// union types live in the type module.
func TestTwoOneofsRenderAsIndependentPresenceUnions(t *testing.T) {
	ts := readGoldenConcat(t, "two_oneofs.ts", "two_oneofs_client.ts")

	// The message type is the intersection of the base and BOTH presence unions.
	if !strings.Contains(ts, "export type TwoOneofs = TwoOneofsBase & TwoOneofsA & TwoOneofsB;") {
		t.Error("expected TwoOneofs to be an intersection of TwoOneofsBase and both oneof unions")
	}

	// Union A: its own arms plus an all-never arm, guarding only its own siblings
	// (x, y) — never the other oneof's keys (p, q).
	unionA := `export type TwoOneofsA =
  | { x: TypeX; y?: never }
  | { y: TypeY; x?: never }
  | { x?: never; y?: never };`
	if !strings.Contains(ts, unionA) {
		t.Errorf("expected TwoOneofsA presence union with only its own sibling guards, got:\n%s", ts)
	}

	// Union B: independent of A — arms + all-never arm guarding only p, q.
	unionB := `export type TwoOneofsB =
  | { p: TypeP; q?: never }
  | { q: TypeQ; p?: never }
  | { p?: never; q?: never };`
	if !strings.Contains(ts, unionB) {
		t.Errorf("expected TwoOneofsB presence union with only its own sibling guards, got:\n%s", ts)
	}

	// The two oneofs are independent: neither union guards against the other's keys.
	if strings.Contains(unionAOf(ts), "p?: never") || strings.Contains(unionAOf(ts), "q?: never") {
		t.Error("TwoOneofsA must not reference oneof B's variant keys (p, q)")
	}
	if strings.Contains(unionBOf(ts), "x?: never") || strings.Contains(unionBOf(ts), "y?: never") {
		t.Error("TwoOneofsB must not reference oneof A's variant keys (x, y)")
	}
}

// unionAOf / unionBOf extract the TwoOneofsA / TwoOneofsB union declaration text so
// cross-oneof leakage can be asserted without matching against the whole file.
func unionAOf(ts string) string {
	return sliceBetween(ts, "export type TwoOneofsA =", "export type TwoOneofsB =")
}

func unionBOf(ts string) string {
	return sliceBetween(ts, "export type TwoOneofsB =", "export interface TwoOneofsBase")
}

func sliceBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return s[i:]
	}
	return s[i : i+j]
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
