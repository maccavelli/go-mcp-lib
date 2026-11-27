package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const draftURI = "https://json-schema.org/draft/2020-12/schema"

var standardVocabs = map[string]bool{
	"core": true, "applicator": true, "unevaluated": true, "validation": true,
	"meta-data": true, "format-annotation": true, "content": true,
}

// walkSchemas visits schema positions, never arbitrary instance-valued keywords
// such as const/enum/default or user property names that resemble keywords.
func walkSchemas(document any, pointer string, visit func(map[string]any, string) error) error {
	object, ok := document.(map[string]any)
	if !ok {
		return nil // booleans and malformed shapes are owned by the metaschema
	}
	if err := visit(object, pointer); err != nil {
		return err
	}
	for _, keyword := range []string{"$defs", "definitions", "properties", "patternProperties", "dependentSchemas"} {
		children, _ := object[keyword].(map[string]any)
		for _, name := range sortedKeys(children) {
			if err := walkSchemas(children[name], pointer+"/"+keyword+"/"+escapePointer(name), visit); err != nil {
				return err
			}
		}
	}
	for _, keyword := range []string{"additionalProperties", "items", "contains", "propertyNames", "unevaluatedProperties", "unevaluatedItems", "not", "if", "then", "else", "contentSchema"} {
		if err := walkSchemas(object[keyword], pointer+"/"+keyword, visit); err != nil {
			return err
		}
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		children, _ := object[keyword].([]any)
		for i, child := range children {
			if err := walkSchemas(child, pointer+"/"+keyword+"/"+strconv.Itoa(i), visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkProfile(document any, canonical string) error {
	return walkSchemas(document, "", func(object map[string]any, pointer string) error {
		location := canonical + "#" + pointer
		return checkSchemaNode(object, location)
	})
}

func checkSchemaNode(object map[string]any, location string) error {
	if dialect, ok := object["$schema"].(string); ok && dialect != draftURI && dialect != draftURI+"#" {
		return fmt.Errorf("validation: %s/$schema: unsupported dialect %q; require %s", location, clip(dialect), draftURI)
	}
	vocabs, _ := object["$vocabulary"].(map[string]any)
	for _, uri := range sortedKeys(vocabs) {
		qualified := strings.HasPrefix(uri, "https://json-schema.org/draft/2020-12/vocab/") && standardVocabs[strings.TrimPrefix(uri, "https://json-schema.org/draft/2020-12/vocab/")]
		if required, _ := vocabs[uri].(bool); required && !qualified {
			return fmt.Errorf("validation: %s/$vocabulary/%s: unqualified required vocabulary", location, escapePointer(uri))
		}
	}
	if pattern, ok := object["pattern"].(string); ok {
		if _, err := compilePattern(pattern); err != nil {
			return RejectUnknownRegexSemantics(pattern, location+"/pattern")
		}
	}
	patterns, _ := object["patternProperties"].(map[string]any)
	for _, pattern := range sortedKeys(patterns) {
		if _, err := compilePattern(pattern); err != nil {
			return RejectUnknownRegexSemantics(pattern, location+"/patternProperties/"+escapePointer(pattern))
		}
	}
	return nil
}

func escapePointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

// compilePattern implements the qualified scalar-Unicode RE2/ECMA common
// subset. ECMA dot excludes all four line terminators. Engine flags, lookaround,
// backreferences, Unicode/whitespace escapes and Go-only escapes are rejected.
func compilePattern(pattern string) (jsonschema.Regexp, error) {
	if strings.Contains(pattern, "[:") {
		return nil, fmt.Errorf("unqualified POSIX character class")
	}
	var translated strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		char := pattern[i]
		if char == '\\' {
			i++
			if i == len(pattern) || strings.ContainsRune("0123456789asSuUpPAzZQEC", rune(pattern[i])) || pattern[i] == 'x' && i+1 < len(pattern) && pattern[i+1] == '{' {
				return nil, fmt.Errorf("unqualified escape")
			}
			translated.WriteByte('\\')
			translated.WriteByte(pattern[i])
			continue
		}
		if char == '(' && i+1 < len(pattern) && pattern[i+1] == '?' && !strings.HasPrefix(pattern[i:], "(?:") {
			return nil, fmt.Errorf("unqualified group or engine flag")
		}
		switch char {
		case '[':
			inClass = true
		case ']':
			inClass = false
		}
		if char == '.' && !inClass {
			translated.WriteString(`[^\n\r\x{2028}\x{2029}]`)
		} else {
			translated.WriteByte(char)
		}
	}
	return regexp.Compile(translated.String())
}
