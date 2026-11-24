package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// exactUnmarshal decodes JSON preserving numeric tokens exactly, in the
// spirit of the framework's lossless contract: numbers decode to
// json.Number-backed values rather than float64, so validation compares
// what the wire said, not what a lossy conversion produced.
func exactUnmarshal(body string, out any) error {
	dec := json.NewDecoder(bytes.NewReader([]byte(body)))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("trailing value after JSON document")
	}
	return nil
}
