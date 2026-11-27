package validation

import (
	"errors"
	"net/url"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type diagnosticError struct {
	message string
	cause   error
}

func (err *diagnosticError) Error() string { return err.message }
func (err *diagnosticError) Unwrap() error { return err.cause }

func shortErr(err error) error {
	message := err.Error()
	var schemaError *jsonschema.SchemaValidationError
	var detail *jsonschema.ValidationError
	if errors.As(err, &schemaError) && errors.As(schemaError.Err, &detail) {
		if locations := schemaErrorLocations(schemaError.URL, detail); len(locations) != 0 {
			message = strings.Join(locations, ", ") + ": " + message
		}
	}
	if len(message) > 300 {
		message = message[:300] + "…"
	}
	return &diagnosticError{message: message, cause: err}
}

// The metaschema validates a referenced schema as its instance. Its leaf
// locations are therefore relative to SchemaValidationError.URL, not to the
// declaration root or to the metaschema URL. Never reconstruct them from prose.
func schemaErrorLocations(location string, detail *jsonschema.ValidationError) []string {
	base, err := url.Parse(location)
	if err != nil {
		return nil
	}
	locations := make([]string, 0, 8)
	var visit func(*jsonschema.ValidationError)
	visit = func(node *jsonschema.ValidationError) {
		if node == nil || len(locations) >= 8 {
			return
		}
		if len(node.Causes) != 0 {
			for _, child := range node.Causes {
				visit(child)
			}
			return
		}
		pointer := base.Fragment
		for _, token := range node.InstanceLocation {
			pointer += "/" + escapePointer(token)
		}
		resolved := *base
		resolved.Fragment, resolved.RawFragment = pointer, ""
		locations = append(locations, resolved.String())
	}
	visit(detail)
	return locations
}
