// Package urlparamtest holds the cross-generator consistency tests for URL
// parameter validation.
//
// The fixtures live here, in one place, rather than being copied into each
// generator's testdata directory. The per-generator copy convention exists so that
// golden files stay independent; these fixtures produce no goldens, and six copies
// of the same proto is exactly how internal/pyclientgen/testdata/proto/query_params.proto
// drifted from its five siblings.
//
// The tests assert that all six generators reject the same protos with the same
// message. A shared helper in internal/annotations is not enough on its own to keep
// them aligned — annotations.ValidateTimestampFormatAnnotation is shared too, and
// only two of the six generators ever wired it in.
package urlparamtest
