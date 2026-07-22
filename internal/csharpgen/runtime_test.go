package csharpgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
)

// TestGeneratedClientsExecute complements compile/golden coverage by running the
// generated clients against a fake HttpMessageHandler. The C# program verifies
// the public runtime contract for both supported JSON libraries.
func TestGeneratedClientsExecute(t *testing.T) {
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

	for _, tc := range []struct {
		name    string
		jsonLib string
	}{
		{name: "newtonsoft", jsonLib: "newtonsoft"},
		{name: "system text json", jsonLib: "system_text_json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputDir := t.TempDir()
			runRuntimeProtoc(t, protoDir, projectRoot, pluginPath, outputDir, tc.jsonLib)
			writeRuntimeProject(t, outputDir, tc.jsonLib)
			if err := os.WriteFile(filepath.Join(outputDir, "Program.cs"), []byte(runtimeProgram), 0o644); err != nil {
				t.Fatalf("write C# runtime test program: %v", err)
			}

			build := exec.Command("dotnet", "build", "Runtime.csproj", "--nologo", "--verbosity", "minimal", "--ignore-failed-sources")
			build.Dir = outputDir
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("generated %s runtime project does not compile: %v\n%s", tc.jsonLib, err, output)
			}

			run := exec.Command("dotnet", "run", "--project", "Runtime.csproj", "--no-build", "--no-restore")
			run.Dir = outputDir
			if output, err := run.CombinedOutput(); err != nil {
				t.Fatalf("generated %s runtime checks failed: %v\n%s", tc.jsonLib, err, output)
			}
		})
	}
}

func runRuntimeProtoc(t *testing.T, protoDir, projectRoot, pluginPath, outputDir, jsonLib string) {
	t.Helper()
	args := []string{
		"--plugin=protoc-gen-csharp-http=" + pluginPath,
		"--csharp-http_out=" + outputDir,
		"--csharp-http_opt=namespace=Test.Contracts,json_lib=" + jsonLib,
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
		"comprehensive_models.proto",
		"runtime_services.proto",
	}
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("runtime protoc generation (%s): %v\n%s", jsonLib, err, stderr.String())
	}
}

func writeRuntimeProject(t *testing.T, outputDir, jsonLib string) {
	t.Helper()
	packageReference := ""
	if jsonLib == "newtonsoft" {
		packageReference = `
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />`
	}
	project := fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <TreatWarningsAsErrors>true</TreatWarningsAsErrors>
    <NuGetAudit>false</NuGetAudit>
  </PropertyGroup>
  <ItemGroup>%s
  </ItemGroup>
</Project>
`, packageReference)
	if err := os.WriteFile(filepath.Join(outputDir, "Runtime.csproj"), []byte(project), 0o644); err != nil {
		t.Fatalf("write C# runtime project: %v", err)
	}
}

const runtimeProgram = `using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using Test.Contracts;

internal static class Program
{
    private static int _requestNumber;

    private static async Task Main()
    {
        await ExerciseSuccessfulRequests();
        await ExerciseErrorDispatch();
        await ExerciseCancellation();
    }

    private static async Task ExerciseSuccessfulRequests()
    {
        _requestNumber = 0;
        using var httpClient = new HttpClient(new DelegateHandler(HandleSuccessfulRequest));
        var client = new RuntimeServiceClient("https://example.test/ignored", new RuntimeServiceClientOptions
        {
            HttpClient = httpClient,
            DefaultHeaders = new Dictionary<string, string>
            {
                ["X-Default"] = "service"
            },
            Service = "present"
        });

        var widget = await client.EchoWidgetAsync(new RuntimeWidgetRequest
        {
            Id = "a/b c",
            OwnerId = "owner value",
            Widget = new Widget
            {
                Id = "body-id",
                Payload = new byte[] { 0x0a, 0xff },
                State = WidgetState.StateReady,
                StateLabels = new Dictionary<string, WidgetState> { ["primary"] = WidgetState.StateReady },
                StateHistory = new List<WidgetState> { WidgetState.StateReady },
                PayloadChunks = new List<byte[]> { new byte[] { 0x0a, 0xff } },
                PayloadsById = new Dictionary<string, byte[]> { ["primary"] = new byte[] { 0x0a, 0xff } },
                DisplayState = WidgetState.StateReady,
                MetaNote = "client-flat"
            }
        }, new RuntimeServiceCallOptions
        {
            Headers = new Dictionary<string, string>
            {
                ["X-Default"] = "call",
                ["X-Call"] = "present"
            },
            RequestId = "request-123"
        });

        Equal("server", widget.Id, "widget response id");
        SequenceEqual(new byte[] { 0x0a, 0xff }, widget.Payload, "scalar hex bytes response");
        Equal(WidgetState.StateReady, widget.State, "numeric enum response");
        Equal(WidgetState.StateReady, widget.StateLabels["primary"], "map enum response");
        Equal(WidgetState.StateReady, widget.StateHistory.Single(), "list enum response");
        SequenceEqual(new byte[] { 0x0a, 0xff }, widget.PayloadChunks.Single(), "list bytes response");
        SequenceEqual(new byte[] { 0x0a, 0xff }, widget.PayloadsById["primary"], "map bytes response");
        Equal(WidgetState.StateReady, widget.DisplayState, "string enum response");
        Equal("server-flat", widget.MetaNote, "flattened response property");

        var search = await client.SearchWidgetsAsync(new RuntimeSearchRequest
        {
            OwnerId = "owner value",
            TagIds = new List<string> { "red/blue", "green" }
        });
        Equal("search", search.Id, "GET query response");

        var shape = await client.EchoShapeAsync(new ShapeEnvelope { Radius = 2.5 });
        Equal("circle_shape", shape.Kind, "oneof discriminator response");
        Equal(3.5, shape.Radius, "flattened oneof response");
        True(shape.Width is null, "unselected oneof variant must be absent");

        var tags = await client.EchoTagsAsync(new TagList { "one", "two" });
        Equal(2, tags.Count, "root unwrap response count");
        Equal("two", tags[1], "root unwrap response value");
        Equal(4, _requestNumber, "successful request count");
    }

    private static async Task<HttpResponseMessage> HandleSuccessfulRequest(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        _requestNumber++;
        Equal("application/json", request.Headers.Accept.Single().MediaType, "Accept header");
        Equal("present", request.Headers.GetValues("X-Service").Single(), "default header");

        switch (_requestNumber)
        {
            case 1:
            {
                Equal(HttpMethod.Patch, request.Method, "widget HTTP verb");
                Equal("/ignored/runtime/v1/widgets/a%2Fb%20c", request.RequestUri!.PathAndQuery, "path binding");
                Equal("call", request.Headers.GetValues("X-Default").Single(), "call header overrides default");
                Equal("present", request.Headers.GetValues("X-Call").Single(), "call header");
                Equal("request-123", request.Headers.GetValues("X-Request-ID").Single(), "typed method header");
                Equal("application/json", request.Content!.Headers.ContentType!.MediaType, "request Content-Type");
                using var body = JsonDocument.Parse(await request.Content.ReadAsStringAsync(cancellationToken));
                var root = body.RootElement;
                Equal("a/b c", root.GetProperty("id").GetString(), "request path field remains in body");
                Equal("owner value", root.GetProperty("owner_id").GetString(), "request query field remains in body");
                var payload = root.GetProperty("widget");
                Equal("body-id", payload.GetProperty("id").GetString(), "nested request body id");
                Equal("client-flat", payload.GetProperty("meta_note").GetString(), "flattened request property");
                Equal("0aff", payload.GetProperty("payload").GetString(), "scalar hex bytes request");
                Equal(1, payload.GetProperty("state").GetInt32(), "numeric enum request");
                Equal("ready", payload.GetProperty("state_labels").GetProperty("primary").GetString(), "map enum request");
                Equal("ready", payload.GetProperty("state_history")[0].GetString(), "list enum request");
                Equal("0aff", payload.GetProperty("payload_chunks")[0].GetString(), "list bytes request");
                Equal("0aff", payload.GetProperty("payloads_by_id").GetProperty("primary").GetString(), "map bytes request");
                Equal("ready", payload.GetProperty("display_state").GetString(), "string enum request");
                return Json(HttpStatusCode.OK, """{"id":"server","payload":"0aff","state":1,"state_labels":{"primary":"ready"},"state_history":["ready"],"payload_chunks":["0aff"],"payloads_by_id":{"primary":"0aff"},"display_state":"ready","meta_note":"server-flat"}""");
            }
            case 2:
            {
                Equal(HttpMethod.Get, request.Method, "query HTTP verb");
                Equal("/ignored/runtime/v1/widgets?owner=owner%20value&tag=red%2Fblue&tag=green", request.RequestUri!.PathAndQuery, "query binding");
                True(request.Content is null, "GET query request has no body");
                return Json(HttpStatusCode.OK, """{"id":"search"}""");
            }
            case 3:
            {
                Equal(HttpMethod.Post, request.Method, "shape HTTP verb");
                Equal("/ignored/runtime/v1/shapes", request.RequestUri!.PathAndQuery, "shape path");
                using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync(cancellationToken));
                Equal("circle_shape", body.RootElement.GetProperty("kind").GetString(), "inferred oneof discriminator request");
                Equal(2.5, body.RootElement.GetProperty("radius").GetDouble(), "flattened oneof request");
                True(!body.RootElement.TryGetProperty("rectangle", out _), "unselected oneof request variant");
                return Json(HttpStatusCode.OK, """{"kind":"circle_shape","radius":3.5}""");
            }
            case 4:
            {
                Equal(HttpMethod.Put, request.Method, "unwrap HTTP verb");
                Equal("/ignored/runtime/v1/tags", request.RequestUri!.PathAndQuery, "unwrap path");
                using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync(cancellationToken));
                Equal(JsonValueKind.Array, body.RootElement.ValueKind, "root unwrap request shape");
                Equal("one", body.RootElement[0].GetString(), "root unwrap request value");
                return Json(HttpStatusCode.OK, """["one","two"]""");
            }
            default:
                throw new InvalidOperationException("Unexpected successful request");
        }
    }

    private static async Task ExerciseErrorDispatch()
    {
        var errorNumber = 0;
        using var httpClient = new HttpClient(new DelegateHandler((_, _) =>
        {
            errorNumber++;
            return Task.FromResult(errorNumber == 1
                ? Json(HttpStatusCode.BadRequest, """{"violations":[{"field":"id","description":"required"}]}""")
                : Json(HttpStatusCode.Conflict, """{"reason":"duplicate","email":"user@example.test","retry_after_seconds":9}""", "trace-123"));
        }));
        var client = new RuntimeServiceClient("https://example.test", new RuntimeServiceClientOptions { HttpClient = httpClient });

        try
        {
            await client.EchoShapeAsync(new ShapeEnvelope());
            throw new InvalidOperationException("Expected ValidationException");
        }
        catch (ValidationException error)
        {
            Equal(400, error.StatusCode, "validation status");
            Equal("id", error.Violations.Single().Field, "validation field");
            Equal("required", error.Violations.Single().Description, "validation description");
        }

        try
        {
            await client.EchoShapeAsync(new ShapeEnvelope());
            throw new InvalidOperationException("Expected LoginErrorException");
        }
        catch (LoginErrorException error)
        {
            Equal(409, error.StatusCode, "typed proto error status");
            Equal("duplicate", error.Reason, "typed proto error payload");
            Equal("user@example.test", error.Email, "typed proto error email");
            Equal(9, error.RetryAfterSeconds, "typed proto error scalar");
            Equal("trace-123", error.Headers["X-Trace"].Single(), "typed proto error response header");
        }
    }

    private static async Task ExerciseCancellation()
    {
        using var httpClient = new HttpClient(new DelegateHandler(async (_, cancellationToken) =>
        {
            await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken);
            throw new InvalidOperationException("Unreachable");
        }));
        var client = new RuntimeServiceClient("https://example.test", new RuntimeServiceClientOptions
        {
            HttpClient = httpClient,
            Timeout = TimeSpan.FromMilliseconds(25)
        });

        await ThrowsCancellation(() => client.EchoShapeAsync(new ShapeEnvelope()), "client timeout");

        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await ThrowsCancellation(
            () => client.EchoShapeAsync(new ShapeEnvelope(), cancellationToken: cancellation.Token),
            "caller cancellation");
    }

    private static HttpResponseMessage Json(HttpStatusCode status, string body, string? trace = null)
    {
        var response = new HttpResponseMessage(status)
        {
            Content = new StringContent(body, Encoding.UTF8, "application/json")
        };
        if (trace is not null)
        {
            response.Headers.Add("X-Trace", trace);
        }
        return response;
    }

    private static async Task ThrowsCancellation(Func<Task> action, string name)
    {
        try
        {
            await action();
            throw new InvalidOperationException($"Expected cancellation: {name}");
        }
        catch (OperationCanceledException)
        {
        }
    }

    private static void True(bool condition, string name)
    {
        if (!condition) throw new InvalidOperationException($"Assertion failed: {name}");
    }

    private static void Equal<T>(T expected, T actual, string name)
    {
        if (!EqualityComparer<T>.Default.Equals(expected, actual))
        {
            throw new InvalidOperationException($"Assertion failed: {name}. Expected '{expected}', actual '{actual}'.");
        }
    }

    private static void SequenceEqual(byte[] expected, byte[] actual, string name)
    {
        if (!expected.SequenceEqual(actual))
        {
            throw new InvalidOperationException($"Assertion failed: {name}. Byte arrays differ.");
        }
    }

    private sealed class DelegateHandler : HttpMessageHandler
    {
        private readonly Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> _handler;

        public DelegateHandler(Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> handler)
        {
            _handler = handler;
        }

        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
            => _handler(request, cancellationToken);
    }
}
`
