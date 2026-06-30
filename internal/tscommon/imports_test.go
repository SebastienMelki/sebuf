package tscommon

import (
	"fmt"
	"strings"
	"testing"
)

func TestRelativeImportSpecifier(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want string
	}{
		{"album/v1/service_client", "core/v1/identifiers", "../../core/v1/identifiers"},
		{"album/v1/service_client", "album/v1/album", "./album"},
		{"album/v1/service_client", "errors", "../../errors"},
		{"service_client", "errors", "./errors"},
		{"service_client", "album", "./album"},
		{"a/b/c_client", "a/b/c", "./c"},
		{"a/b/c/deep_client", "x/y", "../../../x/y"},
	}
	for _, tt := range tests {
		if got := RelativeImportSpecifier(tt.from, tt.to); got != tt.want {
			t.Errorf("RelativeImportSpecifier(%q,%q) = %q, want %q", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestModuleForFile(t *testing.T) {
	if got := ModuleForFile("anghamna/core/v1/identifiers.proto"); got != "anghamna/core/v1/identifiers" {
		t.Errorf("ModuleForFile = %q", got)
	}
}

func TestImportTracker_SameSymbolMemoized(t *testing.T) {
	tr := NewImportTracker()
	a1 := tr.NeedType("./album", "Album")
	a2 := tr.NeedType("./album", "Album")
	if a1 != "Album" || a2 != "Album" {
		t.Fatalf("expected stable alias Album, got %q/%q", a1, a2)
	}
	if got := len(tr.typeImports["./album"]); got != 1 {
		t.Errorf("expected 1 recorded symbol, got %d", got)
	}
}

func TestImportTracker_CollisionAliasing(t *testing.T) {
	tr := NewImportTracker()
	first := tr.NeedType("../a/meta", "Metadata")
	second := tr.NeedType("../b/meta", "Metadata")
	if first != "Metadata" {
		t.Errorf("first reference should keep bare name, got %q", first)
	}
	if second != "Metadata_1" {
		t.Errorf("colliding reference should alias to Metadata_1, got %q", second)
	}
	// memoized on repeat
	if again := tr.NeedType("../b/meta", "Metadata"); again != "Metadata_1" {
		t.Errorf("repeat should return Metadata_1, got %q", again)
	}
}

func TestImportTracker_RenderOrdering(t *testing.T) {
	tr := NewImportTracker()
	tr.NeedErrors("./errors")
	tr.NeedType("../../core/v1/identifiers", "ArtistID")
	tr.NeedType("../../core/v1/identifiers", "AlbumID")
	tr.NeedType("./album", "Album")

	var b strings.Builder
	p := Printer(func(format string, args ...interface{}) {
		b.WriteString(fmt.Sprintf(format, args...))
		b.WriteString("\n")
	})
	tr.Render(p)
	got := b.String()

	want := strings.Join([]string{
		`import { ApiError, FieldViolation, ValidationError } from "./errors";`,
		`import type { AlbumID, ArtistID } from "../../core/v1/identifiers";`,
		`import type { Album } from "./album";`,
		``,
		``,
	}, "\n")
	if got != want {
		t.Errorf("Render mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestImportTracker_Empty(t *testing.T) {
	tr := NewImportTracker()
	if !tr.Empty() {
		t.Error("new tracker should be empty")
	}
	tr.NeedType("./x", "Y")
	if tr.Empty() {
		t.Error("tracker should be non-empty after NeedType")
	}
}
