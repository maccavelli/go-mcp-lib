package validation

import "github.com/maccavelli/go-mcp-lib/codec"

// exactUnmarshal decodes JSON preserving numeric tokens exactly, in the
// spirit of the framework's lossless contract: numbers decode to
// json.Number-backed values rather than float64, so validation compares
// what the wire said, not what a lossy conversion produced.
func exactUnmarshal(body string, out any) error {
	return codec.Unmarshal([]byte(body), out, codec.DefaultBounds())
}
