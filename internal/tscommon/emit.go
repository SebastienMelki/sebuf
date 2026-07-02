package tscommon

// EmitContext threads plugin options through the (otherwise stateless) type
// emitters. A nil context — or one with the default options — produces output
// byte-identical to the historical behavior.
type EmitContext struct {
	Options Options
}

// oneofDiscriminated reports whether un-annotated oneofs should render as
// discriminated unions instead of flattened optional fields.
func (c *EmitContext) oneofDiscriminated() bool {
	return c != nil && c.Options.OneofStyle == OneofStyleDiscriminated
}
