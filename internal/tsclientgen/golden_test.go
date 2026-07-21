package tsclientgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTSClientGenGoldenFiles tests TypeScript client generation against golden files.
// This ensures any changes to code generation are intentional and reviewed.
//
// To update golden files after intentional changes:
//
//	UPDATE_GOLDEN=1 go test -run TestTSClientGenGoldenFiles
func TestTSClientGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	testCases := []struct {
		name          string
		protoFile     string
		expectedFiles []string
	}{
		{
			name:      "comprehensive HTTP verbs",
			protoFile: "http_verbs_comprehensive.proto",
			expectedFiles: []string{
				"http_verbs_comprehensive_client.ts",
			},
		},
		{
			name:      "query parameters",
			protoFile: "query_params.proto",
			expectedFiles: []string{
				"query_params_client.ts",
			},
		},
		{
			name:      "backward compatibility",
			protoFile: "backward_compat.proto",
			expectedFiles: []string{
				"backward_compat_client.ts",
			},
		},
		{
			name:      "complex features",
			protoFile: "complex_features.proto",
			expectedFiles: []string{
				"complex_features_client.ts",
			},
		},
		{
			name:      "unwrap variants",
			protoFile: "unwrap.proto",
			expectedFiles: []string{
				"unwrap_client.ts",
			},
		},
		{
			name:      "int64 encoding",
			protoFile: "int64_encoding.proto",
			expectedFiles: []string{
				"int64_encoding_client.ts",
			},
		},
		{
			name:      "enum encoding",
			protoFile: "enum_encoding.proto",
			expectedFiles: []string{
				"enum_encoding_client.ts",
			},
		},
		{
			name:      "nullable fields",
			protoFile: "nullable.proto",
			expectedFiles: []string{
				"nullable_client.ts",
			},
		},
		{
			name:      "empty behavior",
			protoFile: "empty_behavior.proto",
			expectedFiles: []string{
				"empty_behavior_client.ts",
			},
		},
		{
			name:      "timestamp format",
			protoFile: "timestamp_format.proto",
			expectedFiles: []string{
				"timestamp_format_client.ts",
			},
		},
		{
			name:      "bytes encoding",
			protoFile: "bytes_encoding.proto",
			expectedFiles: []string{
				"bytes_encoding_client.ts",
			},
		},
		{
			name:      "flatten",
			protoFile: "flatten.proto",
			expectedFiles: []string{
				"flatten_client.ts",
			},
		},
		{
			name:      "oneof discriminator",
			protoFile: "oneof_discriminator.proto",
			expectedFiles: []string{
				"oneof_discriminator_client.ts",
			},
		},
		{
			name:      "multi-word oneof name",
			protoFile: "multi_word_oneof.proto",
			expectedFiles: []string{
				"multi_word_oneof_client.ts",
			},
		},
		{
			name:      "two un-annotated oneofs in one message",
			protoFile: "two_oneofs.proto",
			expectedFiles: []string{
				"two_oneofs_client.ts",
			},
		},
		{
			name:      "un-annotated oneof with enum and timestamp variants",
			protoFile: "oneof_field_typing.proto",
			expectedFiles: []string{
				"oneof_field_typing_client.ts",
			},
		},
		{
			name:      "flatten oneof unset arm guards child keys",
			protoFile: "flatten_oneof_unset.proto",
			expectedFiles: []string{
				"flatten_oneof_unset_client.ts",
			},
		},
		{
			name:      "SSE streaming",
			protoFile: "sse.proto",
			expectedFiles: []string{
				"sse_client.ts",
			},
		},
		{
			name:      "empty request body",
			protoFile: "empty_request_body.proto",
			expectedFiles: []string{
				"empty_request_body_client.ts",
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

	mkdirErr := os.MkdirAll(goldenDir, 0o755)
	if mkdirErr != nil {
		t.Fatalf("Failed to create golden directory: %v", mkdirErr)
	}

	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-ts-client")

	// Build the plugin if it doesn't exist
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	tempDir := t.TempDir()

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			protoPath := filepath.Join(protoDir, tc.protoFile)

			_, statErr := os.Stat(protoPath)
			if os.IsNotExist(statErr) {
				t.Fatalf("Proto file not found: %s", protoPath)
			}

			// Run protoc with ts-client plugin
			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-ts-client="+pluginPath,
				"--ts-client_out="+tempDir,
				"--ts-client_opt=paths=source_relative",
				"--proto_path="+protoDir,
				"--proto_path="+filepath.Join(projectRoot, "proto"),
				tc.protoFile,
			)
			cmd.Dir = protoDir

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			runErr := cmd.Run()
			if runErr != nil {
				t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
			}

			for _, expectedFile := range tc.expectedFiles {
				generatedPath := filepath.Join(tempDir, expectedFile)
				goldenPath := filepath.Join(goldenDir, expectedFile)

				generatedContent, readErr := os.ReadFile(generatedPath)
				if readErr != nil {
					t.Fatalf("Failed to read generated file %s: %v", generatedPath, readErr)
				}

				if updateGolden {
					updateGoldenFile(t, goldenPath, generatedContent)
					continue
				}
				compareGoldenFile(t, expectedFile, goldenPath, generatedContent)
			}
		})
	}
}

// TestMultiWordOneofNameDoesNotLeak asserts the regression fixed on this branch:
// a multi-word oneof name (super_title_image) must surface only as the PascalCase
// union type name and never leak into the generated TypeScript as a raw
// snake_case wrapper property. See internal/tscommon/types.go
// (GenerateOneofDiscriminatedUnionType / GenerateStandardInterface).
func TestMultiWordOneofNameDoesNotLeak(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	goldenPath := filepath.Join(wd, "testdata", "golden", "multi_word_oneof_client.ts")

	content, readErr := os.ReadFile(goldenPath)
	if readErr != nil {
		t.Fatalf("Failed to read golden file %s: %v", goldenPath, readErr)
	}
	ts := string(content)

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
// neither union references the other's variant keys.
func TestTwoOneofsRenderAsIndependentPresenceUnions(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	goldenPath := filepath.Join(wd, "testdata", "golden", "two_oneofs_client.ts")

	content, readErr := os.ReadFile(goldenPath)
	if readErr != nil {
		t.Fatalf("Failed to read golden file %s: %v", goldenPath, readErr)
	}
	ts := string(content)

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
	writeErr := os.WriteFile(goldenPath, content, 0o644)
	if writeErr != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, writeErr)
	}
	t.Logf("Updated golden file: %s", goldenPath)
}

func compareGoldenFile(t *testing.T, expectedFile, goldenPath string, generatedContent []byte) {
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
			expectedFile,
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
