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

func TestParseErrorHandlingOption(t *testing.T) {
	if got := ParseErrorHandling(""); got != ErrorHandlingThrow {
		t.Fatalf("default = %v, want throw", got)
	}
	if got := ParseErrorHandling("ts_error_handling=throw"); got != ErrorHandlingThrow {
		t.Fatalf("explicit throw = %v, want throw", got)
	}
	if got := ParseErrorHandling("ts_runtime=protobuf-es,ts_error_handling=result"); got != ErrorHandlingResult {
		t.Fatalf("result = %v, want result", got)
	}
}

func TestValidateRuntimeOptions(t *testing.T) {
	// result requires es — reject every non-es runtime.
	if err := ValidateRuntimeOptions(MessageRuntimeHandRolled, ErrorHandlingResult); err == nil {
		t.Fatal("expected error for ts_error_handling=result without ts_runtime=protobuf-es")
	}
	// Valid combinations.
	for _, tc := range []struct {
		runtime MessageRuntime
		errh    ErrorHandling
	}{
		{MessageRuntimeHandRolled, ErrorHandlingThrow},
		{MessageRuntimeES, ErrorHandlingThrow},
		{MessageRuntimeES, ErrorHandlingResult},
	} {
		if err := ValidateRuntimeOptions(tc.runtime, tc.errh); err != nil {
			t.Errorf("ValidateRuntimeOptions(%v, %v) = %v, want nil", tc.runtime, tc.errh, err)
		}
	}
}
