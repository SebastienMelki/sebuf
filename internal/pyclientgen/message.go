package pyclientgen

import (
	"google.golang.org/protobuf/compiler/protogen"
)

// writeMessage emits a Python @dataclass for a proto message.
//
// Body is intentionally minimal in this scaffold commit; subsequent commits
// add field rendering, JSON-mapping annotation handling (int64/enum/bytes/
// timestamp/nullable/empty_behavior/unwrap/flatten/oneof), and to_dict /
// from_dict helpers.
func writeMessage(p printer, msg *protogen.Message) {
	className := pythonTypeName(msg)
	p("@dataclass")
	p("class %s:", className)
	p(`    """Generated from proto message %s."""`, msg.Desc.FullName())
	if len(msg.Fields) == 0 {
		p("    pass")
	} else {
		// Minimal placeholder — real field rendering lands in the next commit.
		for _, f := range msg.Fields {
			name := escapePyKeyword(string(f.Desc.Name()))
			p("    %s: Any = None  # TODO: real type", name)
		}
	}
	p("")
	p("")
}
