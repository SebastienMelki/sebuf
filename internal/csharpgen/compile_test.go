package csharpgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
)

// TestGeneratedContractsCompile exercises the output through the C# compiler.
// Golden files catch unintended text changes, but cannot catch an invalid using,
// attribute, generic constraint, or package reference.  Keep this test based on
// protoc output rather than the committed goldens so it always validates the
// current plugin binary.
func TestGeneratedContractsCompile(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found; generated C# compile validation requires protoc")
	}
	if _, err := exec.LookPath("dotnet"); err != nil {
		t.Skip("dotnet SDK not found; generated C# compile validation requires dotnet")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectRoot := filepath.Join(workingDir, "..", "..")
	protoDir := filepath.Join(workingDir, "testdata", "proto")
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-csharp-http")

	for _, tc := range []struct {
		name    string
		jsonLib string
	}{
		{name: "newtonsoft", jsonLib: "newtonsoft"},
		{name: "system text json", jsonLib: "system_text_json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputDir := t.TempDir()
			runProtoc(t, protoDir, projectRoot, pluginPath, outputDir, tc.jsonLib)
			assertCrossPackageOutput(t, outputDir)
			writeCompileProject(t, outputDir, tc.jsonLib)

			cmd := exec.Command("dotnet", "build", "Compile.csproj", "--nologo", "--verbosity", "minimal", "--ignore-failed-sources")
			cmd.Dir = outputDir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated %s contracts do not compile: %v\n%s", tc.jsonLib, err, output)
			}
		})
	}
}

func assertCrossPackageOutput(t *testing.T, outputDir string) {
	t.Helper()
	alphaPath := filepath.Join(outputDir, "test", "alpha", "v1", "Contracts.g.cs")
	alphaBytes, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatalf("read generated alpha contracts: %v", err)
	}
	alpha := string(alphaBytes)
	for _, want := range []string{
		"namespace Test.Contracts.Test.Alpha.V1",
		"public Shared? Local { get; set; }",
		"public global::Test.Contracts.Test.Beta.V1.Shared? Remote { get; set; }",
		"Dictionary<string, global::Test.Contracts.Test.Beta.V1.Shared>",
		"Task<global::Test.Contracts.Test.Beta.V1.Reply> RelayAsync(global::Test.Contracts.Test.Beta.V1.Shared req",
	} {
		if !strings.Contains(alpha, want) {
			t.Errorf("generated alpha contracts do not contain %q", want)
		}
	}

	betaPath := filepath.Join(outputDir, "test", "beta", "v1", "Contracts.g.cs")
	betaBytes, err := os.ReadFile(betaPath)
	if err != nil {
		t.Fatalf("read generated beta contracts: %v", err)
	}
	if beta := string(betaBytes); !strings.Contains(beta, "namespace Test.Contracts.Test.Beta.V1") {
		t.Errorf("generated beta contracts use an unexpected namespace")
	}
}

func runProtoc(t *testing.T, protoDir, projectRoot, pluginPath, outputDir, jsonLib string) {
	t.Helper()
	args := []string{
		"--plugin=protoc-gen-csharp-http=" + pluginPath,
		"--csharp-http_out=" + outputDir,
		"--csharp-http_opt=namespace=Test.Contracts,json_lib=" + jsonLib,
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
		"comprehensive_models.proto",
		"comprehensive_services.proto",
		"collision_beta.proto",
		"collision_alpha.proto",
	}
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("protoc generation (%s): %v\n%s", jsonLib, err, stderr.String())
	}
}

func writeCompileProject(t *testing.T, outputDir, jsonLib string) {
	t.Helper()
	packageReference := ""
	if jsonLib == "newtonsoft" {
		packageReference = `
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />`
	}
	project := fmt.Sprintf(`<!-- Generated-contract compile test; intentionally no application code. -->
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <TreatWarningsAsErrors>true</TreatWarningsAsErrors>
    <!-- Compilation should not depend on NuGet's vulnerability-feed availability. -->
    <NuGetAudit>false</NuGetAudit>
  </PropertyGroup>
  <ItemGroup>%s
  </ItemGroup>
</Project>
`, packageReference)
	if err := os.WriteFile(filepath.Join(outputDir, "Compile.csproj"), []byte(project), 0o644); err != nil {
		t.Fatalf("write C# compile project: %v", err)
	}
}
