package csharpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/testutil"
	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
)

// TestWirePresenceContract pins how the Newtonsoft output serializes the four
// shapes a hand-built wire contract depends on (an unset message field, an
// unset optional scalar, an explicit JSON null and an empty repeated field) and
// proves that (sebuf.http.enum_value) keeps an enum's JSON as the given string.
//
// The generated file is compared with a golden; the generated classes are then
// compiled and exercised with JsonConvert, which is how a caller that builds a
// payload without the generated client serializes them.
//
// To update the golden after an intentional change:
//
//	UPDATE_GOLDEN=1 go test -run TestWirePresenceContract
func TestWirePresenceContract(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found; wire presence validation requires protoc")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectRoot := filepath.Join(workingDir, "..", "..")
	protoDir := filepath.Join(workingDir, "testdata", "proto")
	goldenPath := filepath.Join(workingDir, "testdata", "golden", "WirePresence.Newtonsoft.g.cs")
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-csharp-http")

	outputDir := t.TempDir()
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-csharp-http="+pluginPath,
		"--csharp-http_out="+outputDir,
		"--csharp-http_opt=namespace=Test.Wire,json_lib=newtonsoft",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"wire_presence.proto",
	)
	cmd.Dir = protoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	generatedPath := filepath.Join(outputDir, "testdata", "wirepresence", "Contracts.g.cs")
	generated, readErr := os.ReadFile(generatedPath)
	if readErr != nil {
		t.Fatalf("read generated file %s: %v", generatedPath, readErr)
	}
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if writeErr := os.WriteFile(goldenPath, generated, 0o644); writeErr != nil {
			t.Fatalf("write golden %s: %v", goldenPath, writeErr)
		}
	} else {
		golden, goldenErr := os.ReadFile(goldenPath)
		if goldenErr != nil {
			t.Fatalf("read golden %s: %v", goldenPath, goldenErr)
		}
		if !bytes.Equal(generated, golden) {
			t.Fatalf("generated file does not match %s.\nDiff:\n%s",
				goldenPath, testutil.DiffStrings(string(golden), string(generated)))
		}
	}

	if _, lookErr := exec.LookPath("dotnet"); lookErr != nil {
		t.Skip("dotnet SDK not found; the serialization checks need dotnet")
	}

	runDir := t.TempDir()
	if copyErr := os.WriteFile(filepath.Join(runDir, "Contracts.g.cs"), generated, 0o644); copyErr != nil {
		t.Fatalf("copy generated file: %v", copyErr)
	}
	writeRuntimeProject(t, runDir, "newtonsoft")
	if writeErr := os.WriteFile(filepath.Join(runDir, "Program.cs"), []byte(wirePresenceProgram), 0o644); writeErr != nil {
		t.Fatalf("write C# program: %v", writeErr)
	}

	build := exec.Command(
		"dotnet",
		"build",
		"Runtime.csproj",
		"--nologo",
		"--verbosity",
		"minimal",
		"--ignore-failed-sources",
	)
	build.Dir = runDir
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("generated wire presence project does not compile: %v\n%s", buildErr, output)
	}

	run := exec.Command("dotnet", "run", "--project", "Runtime.csproj", "--no-build", "--no-restore")
	run.Dir = runDir
	// A machine that has only a newer .NET runtime than the net8.0 target
	// still runs the program; the checks do not depend on runtime features.
	run.Env = append(os.Environ(), "DOTNET_ROLL_FORWARD=Major")
	if output, runErr := run.CombinedOutput(); runErr != nil {
		t.Fatalf("wire presence checks failed: %v\n%s", runErr, output)
	}
}

// wirePresenceProgram holds the pinned behaviour. Every expected string is what
// the current generator produces; a change to one of them is a change to the
// wire a caller sends and must be a decision, not a side effect.
const wirePresenceProgram = `using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using Test.Wire;

internal static class Program
{
    // The settings a caller needs to put the enum_value strings on the wire.
    private static readonly JsonSerializerSettings WireSettings = new()
    {
        Converters = { new StringEnumConverter() },
        NullValueHandling = NullValueHandling.Ignore,
    };

    private static readonly JsonSerializerSettings WireSettingsKeepNulls = new()
    {
        Converters = { new StringEnumConverter() },
    };

    private static void Main()
    {
        EnumValueKeepsTheGivenString();
        UnsetShapesWithDefaultSettings();
        UnsetShapesWithWireSettings();
        ExplicitNullAndAbsentOnRead();
        Console.WriteLine("ok");
    }

    private static void EnumValueKeepsTheGivenString()
    {
        var probe = new PresenceProbe
        {
            Kind = FileKind.FileKindJpeg,
            Kinds = new List<FileKind> { FileKind.FileKindPng, FileKind.FileKindMp4 },
            OptionalKind = FileKind.FileKindMp4,
        };
        var json = JsonConvert.SerializeObject(probe, WireSettings);
        Equal(
            """{"name":"","tags":[],"places":[],"kind":"JPEG","kinds":["PNG","MP4"],"optional_kind":"MP4"}""",
            json,
            "enum_value strings on the wire (scalar, repeated, optional)");

        // The value without enum_value keeps the proto name.
        var unspecified = JsonConvert.SerializeObject(new PresenceProbe(), WireSettings);
        True(unspecified.Contains("\"kind\":\"FILE_KIND_UNSPECIFIED\""), "unannotated value keeps its proto name");

        // Reading takes the enum_value strings, and the proto name of an annotated value is not accepted.
        var read = JsonConvert.DeserializeObject<PresenceProbe>(
            """{"kind":"PNG","kinds":["JPEG","MP4"],"optional_kind":"JPEG"}""", WireSettings)!;
        Equal(FileKind.FileKindPng, read.Kind, "read scalar enum_value");
        Equal(2, read.Kinds.Count, "read repeated enum_value count");
        Equal(FileKind.FileKindMp4, read.Kinds[1], "read repeated enum_value");
        Equal(FileKind.FileKindJpeg, read.OptionalKind, "read optional enum_value");
        Throws(() => JsonConvert.DeserializeObject<PresenceProbe>("""{"kind":"FILE_KIND_JPEG"}""", WireSettings),
            "the proto name of an annotated value is not on the wire");

        // GAP, pinned: without StringEnumConverter the plain serializer writes the number. Only the
        // generated client normalizes enum values at its boundary, so a caller that serializes a
        // generated class itself must pass the converter.
        var plain = JsonConvert.SerializeObject(probe);
        True(plain.Contains("\"kind\":1,"), "plain JsonConvert writes the enum number");
    }

    private static void UnsetShapesWithDefaultSettings()
    {
        // Default Newtonsoft settings: an unset message field, an unset optional scalar and a
        // nullable scalar all write an explicit null; an empty repeated field writes []; a
        // non-optional scalar writes its default.
        Equal(
            """{"name":"","note":null,"quality":null,"ratio":null,"flag":null,"location":null,"extra":null,"tags":[],"places":[],"kind":0,"kinds":[],"optional_kind":null,"nullable_note":null,"nullable_quality":null,"null_when_empty":null}""",
            JsonConvert.SerializeObject(new PresenceProbe()),
            "default settings, nothing set");
    }

    private static void UnsetShapesWithWireSettings()
    {
        // NullValueHandling.Ignore drops every unset optional scalar and message field. Nothing in
        // the generated type tells an unset field from an explicit null, so a caller that must send
        // a null chooses the settings per payload.
        Equal(
            """{"name":"","tags":[],"places":[],"kind":"FILE_KIND_UNSPECIFIED","kinds":[]}""",
            JsonConvert.SerializeObject(new PresenceProbe(), WireSettings),
            "wire settings, nothing set");

        // An empty repeated field is written as [] unless the list itself is null, which the
        // Ignore setting then drops: that is how a caller keeps a list absent.
        var absentLists = new PresenceProbe { Tags = null!, Places = null!, Kinds = null! };
        Equal(
            """{"name":"","kind":"FILE_KIND_UNSPECIFIED"}""",
            JsonConvert.SerializeObject(absentLists, WireSettings),
            "null lists are absent under Ignore");

        // A set message field and set optional scalars (zero values included) are written.
        var set = new PresenceProbe
        {
            Quality = 0,
            Flag = false,
            Location = new PresenceInner { City = "Paris" },
            Extra = new PresenceInner { City = "", State = "" },
            NullableNote = "n",
        };
        Equal(
            """{"name":"","quality":0,"flag":false,"location":{"city":"Paris"},"extra":{"city":"","state":""},"tags":[],"places":[],"kind":"FILE_KIND_UNSPECIFIED","kinds":[],"nullable_note":"n"}""",
            JsonConvert.SerializeObject(set, WireSettings),
            "set values");

        // The same object with the null-keeping settings shows the nulls again.
        True(JsonConvert.SerializeObject(new PresenceProbe(), WireSettingsKeepNulls).Contains("\"location\":null"),
            "keep-null settings write an unset message field as null");
    }

    private static void ExplicitNullAndAbsentOnRead()
    {
        var absent = JsonConvert.DeserializeObject<PresenceProbe>("{}", WireSettings)!;
        True(absent.Location is null, "absent message field reads as null");
        True(absent.Note is null, "absent optional scalar reads as null");
        Equal(0, absent.Tags.Count, "absent repeated field reads as an empty list");
        Equal(FileKind.FileKindUnspecified, absent.Kind, "absent enum reads as the zero value");

        var nulls = JsonConvert.DeserializeObject<PresenceProbe>(
            """{"location":null,"extra":null,"note":null,"nullable_note":null,"null_when_empty":null,"tags":null,"optional_kind":null}""",
            WireSettings)!;
        True(nulls.Location is null && nulls.Extra is null, "explicit null message field reads as null, the same as absent");
        True(nulls.Note is null && nulls.NullableNote is null, "explicit null scalar reads as null, the same as absent");
        True(nulls.OptionalKind is null, "explicit null optional enum reads as null");
        // An explicit null for a repeated field leaves the initial empty list in place.
        True(nulls.Tags is not null && nulls.Tags.Count == 0, "explicit null repeated field reads as an empty list");

        // Unknown members are ignored on read.
        var extra = JsonConvert.DeserializeObject<PresenceProbe>("""{"name":"a","surprise":1}""", WireSettings)!;
        Equal("a", extra.Name, "unknown member ignored");
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

    private static void Throws(Action action, string name)
    {
        try
        {
            action();
        }
        catch (JsonException)
        {
            return;
        }
        throw new InvalidOperationException($"Assertion failed: expected a JsonException: {name}");
    }
}
`
