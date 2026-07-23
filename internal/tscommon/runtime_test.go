package tscommon

import "testing"

func TestParseTSRuntimeOption(t *testing.T) {
	if got := ParseMessageRuntime(""); got != MessageRuntimeHandRolled {
		t.Fatalf("default = %v, want hand-rolled", got)
	}
	if got := ParseMessageRuntime("ts_runtime=hand-rolled"); got != MessageRuntimeHandRolled {
		t.Fatalf("explicit hand-rolled = %v, want hand-rolled", got)
	}
	if got := ParseMessageRuntime("ts_runtime=protobuf-es"); got != MessageRuntimeES {
		t.Fatalf("es = %v, want protobuf-es", got)
	}
}
