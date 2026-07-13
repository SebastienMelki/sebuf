package tscommon

import "testing"

func TestParseTSRuntimeOption(t *testing.T) {
	if got := parseMessageRuntime(""); got != MessageRuntimeHandRolled {
		t.Fatalf("default = %v, want hand-rolled", got)
	}
	if got := parseMessageRuntime("ts_runtime=protobuf-es"); got != MessageRuntimeES {
		t.Fatalf("es = %v, want protobuf-es", got)
	}
}
