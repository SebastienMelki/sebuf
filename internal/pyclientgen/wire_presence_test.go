package pyclientgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestWirePresenceContract pins how the Python client serializes the four
// shapes a hand-built wire contract depends on (an unset message field, an
// unset optional scalar, an explicit JSON null and an empty repeated field)
// and proves that (sebuf.http.enum_value) keeps an enum's JSON as the given
// string. It generates with the option field_names=proto.
//
// To update the golden after an intentional change:
//
//	UPDATE_GOLDEN=1 go test -run TestWirePresenceContract
func TestWirePresenceContract(t *testing.T) {
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
		"--py-client_opt=paths=source_relative,field_names=proto",
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

	const generatedName = "wire_presence_client.py"
	modulePath := filepath.Join(outDir, generatedName)
	generated, readErr := os.ReadFile(modulePath)
	if readErr != nil {
		t.Fatalf("read generated client: %v", readErr)
	}
	goldenPath := filepath.Join(baseDir, "testdata", "golden", generatedName)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		updateGoldenFile(t, goldenPath, generated)
	} else {
		compareGoldenFile(t, generatedName, goldenPath, generated)
	}

	got := pythonRoundTrip(t, modulePath, wirePresenceSnippet)
	if got != "ok" {
		t.Fatalf("wire presence checks printed %q", got)
	}
}

// wirePresenceSnippet holds the pinned behaviour. Every expected value is what
// the current generator produces; a change to one is a change to the wire a
// caller sends and must be a decision, not a side effect.
const wirePresenceSnippet = `
def dumps(o):
    return json.dumps(o, sort_keys=True, separators=(',', ':'))

# enum_value keeps the given string, for a scalar, a repeated and an optional field.
p = m.PresenceProbe(kind=m.FileKind.FILE_KIND_JPEG,
                    kinds=[m.FileKind.FILE_KIND_PNG, m.FileKind.FILE_KIND_MP4],
                    optional_kind=m.FileKind.FILE_KIND_MP4)
d = p.to_dict()
assert d['kind'] == 'JPEG', d
assert d['kinds'] == ['PNG', 'MP4'], d
assert d['optional_kind'] == 'MP4', d
back = m.PresenceProbe.from_dict(json.loads(dumps(d)))
assert back.kind == m.FileKind.FILE_KIND_JPEG
assert back.kinds == [m.FileKind.FILE_KIND_PNG, m.FileKind.FILE_KIND_MP4]
assert back.optional_kind == m.FileKind.FILE_KIND_MP4
# The proto name of an annotated value is still read (the decoder falls back to it).
assert m.PresenceProbe.from_dict({'kind': 'FILE_KIND_JPEG'}).kind == m.FileKind.FILE_KIND_JPEG
# A value without enum_value keeps its proto name.
assert m.PresenceProbe().to_dict()['kind'] == 'FILE_KIND_UNSPECIFIED'

# Nothing set: the optional scalars, the message fields and the empty repeated
# fields are absent (an empty repeated field cannot be written as []); the
# non-optional scalar and the enum are always written; the nullable fields and
# the empty_behavior=NULL message field are written as null.
assert dumps(m.PresenceProbe().to_dict()) == (
    '{"kind":"FILE_KIND_UNSPECIFIED","name":"","null_when_empty":null,"nullable_note":null,"nullable_quality":null}'
), dumps(m.PresenceProbe().to_dict())

# Set values, zero values included, are written; a set nullable field is written as its value.
s = m.PresenceProbe(quality=0, flag=False, ratio=0.0, note='',
                    location=m.PresenceInner(city='Paris'),
                    extra=m.PresenceInner(city='', state=''),
                    tags=['a'], places=[m.PresenceInner(city='x', state=None)],
                    nullable_note='n', nullable_quality=0)
assert dumps(s.to_dict()) == (
    '{"extra":{"city":"","state":""},"flag":false,"kind":"FILE_KIND_UNSPECIFIED","location":{"city":"Paris"},'
    '"name":"","note":"","null_when_empty":null,"nullable_note":"n","nullable_quality":0,'
    '"places":[{"city":"x"}],"quality":0,"ratio":0.0,"tags":["a"]}'
), dumps(s.to_dict())

# Reading: absent and explicit null are the same (the field keeps its default).
absent = m.PresenceProbe.from_dict({})
nulls = m.PresenceProbe.from_dict({'location': None, 'extra': None, 'note': None, 'nullable_note': None,
                                   'null_when_empty': None, 'tags': None, 'places': None,
                                   'optional_kind': None, 'kind': None})
assert absent == nulls, (absent, nulls)
assert absent.location is None and absent.note is None and absent.tags == [] and absent.optional_kind is None
assert absent.kind == m.FileKind.FILE_KIND_UNSPECIFIED
# A null message object reads as the default message.
assert m.PresenceProbe.from_dict(None) == m.PresenceProbe()
# Unknown members are ignored.
assert m.PresenceProbe.from_dict({'name': 'a', 'surprise': 1}).name == 'a'
print('ok')
`
