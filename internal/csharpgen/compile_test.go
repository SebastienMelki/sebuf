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

func TestGeneratedWireFormattingIsInvariant(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found; generated C# runtime validation requires protoc")
	}
	if _, err := exec.LookPath("dotnet"); err != nil {
		t.Skip("dotnet SDK not found; generated C# runtime validation requires dotnet")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectRoot := filepath.Join(workingDir, "..", "..")
	protoDir := filepath.Join(workingDir, "testdata", "proto")
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-csharp-http")
	outputDir := t.TempDir()
	args := []string{
		"--plugin=protoc-gen-csharp-http=" + pluginPath,
		"--csharp-http_out=" + outputDir,
		"--csharp-http_opt=namespace=Test.Wire,json_lib=system_text_json",
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
		"wire_formatting.proto",
	}
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc generation: %v\n%s", err, output)
	}
	writeCompileProject(t, outputDir, "system_text_json")
	writeWireFormattingProgram(t, outputDir)
	cmd = exec.Command("dotnet", "restore", "Compile.csproj", "--nologo", "--ignore-failed-sources")
	cmd.Dir = outputDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restore generated invariant wire-formatting runtime test: %v\n%s", err, output)
	}
	cmd = exec.Command("dotnet", "run", "--project", "Compile.csproj", "--no-restore", "--nologo", "--verbosity", "minimal")
	cmd.Dir = outputDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated invariant wire-formatting runtime test failed: %v\n%s", err, output)
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

func writeWireFormattingProgram(t *testing.T, outputDir string) {
	t.Helper()
	projectPath := filepath.Join(outputDir, "Compile.csproj")
	project, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read runtime compile project: %v", err)
	}
	project = bytes.Replace(project, []byte("<TargetFramework>net8.0</TargetFramework>"), []byte("<TargetFramework>net8.0</TargetFramework>\n    <OutputType>Exe</OutputType>\n    <UseAppHost>false</UseAppHost>"), 1)
	if err := os.WriteFile(projectPath, project, 0o644); err != nil {
		t.Fatalf("set runtime compile project output type: %v", err)
	}
	const program = `using System;
using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Test.Wire;

sealed class CaptureHandler : HttpMessageHandler
{
    public Uri? RequestUri { get; private set; }
    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        RequestUri = request.RequestUri;
        return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
        {
            Content = new StringContent("{}")
        });
    }
}

static class Program
{
    static async Task Main()
    {
        CultureInfo.CurrentCulture = CultureInfo.GetCultureInfo("de-DE");
        var handler = new CaptureHandler();
        var client = new WireServiceClient("https://example.test", new WireServiceClientOptions
        {
            HttpClient = new HttpClient(handler)
        });
        await client.GetWireAsync(new GetWireRequest
        {
            Mode = Mode.ModeReady,
            NumericMode = Mode.ModeReady,
            Enabled = true,
            Ratio = 1.5f,
            MaxU32 = uint.MaxValue,
            MaxU64 = ulong.MaxValue
        });
        const string expected = "/wire/ready-wire?mode=ready-wire&numeric_mode=1&enabled=true&ratio=1.5&max_u32=4294967295&max_u64=18446744073709551615";
        if (handler.RequestUri?.PathAndQuery != expected)
        {
            throw new InvalidOperationException($"Expected {expected}, got {handler.RequestUri?.PathAndQuery}");
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(outputDir, "Program.cs"), []byte(program), 0o644); err != nil {
		t.Fatalf("write invariant wire-formatting program: %v", err)
	}
}
