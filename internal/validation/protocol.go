package validation

import (
	"crypto/sha256"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

const pinnedProtocolSHA = "907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551"

// RegisterProtocol registers the exact reviewed MCP input and its mandated
// location rules. Merely reusing a URI or a format name cannot enable them.
func (a *Adapter) RegisterProtocol(canonical string, body []byte) error {
	if fmt.Sprintf("%x", sha256.Sum256(body)) != pinnedProtocolSHA {
		return fmt.Errorf("validation: unreviewed protocol schema identity")
	}
	var document any
	if err := exactUnmarshal(string(body), &document); err != nil {
		return err
	}
	owned, err := snapshot(document)
	if err != nil {
		return err
	}
	owned.protocol = true
	return a.register(canonical, owned)
}

type semanticRule struct {
	name  string
	check func(string) error
}

func (rule semanticRule) Validate(ctx *jsonschema.ValidatorContext, value any) {
	text, ok := value.(string)
	if !ok {
		return // the schema's type assertion owns non-string values
	}
	if err := rule.check(text); err != nil {
		ctx.AddError(&kind.Format{Got: value, Want: rule.name, Err: err})
	}
}

func protocolRule(pointer, format string) semanticRule {
	if pointer == "/$defs/Root/properties/uri" {
		return semanticRule{"file URI", CheckRootURI}
	}
	switch format {
	case "uri":
		return semanticRule{format, CheckURI}
	case "uri-template":
		return semanticRule{format, CheckURITemplate}
	case "byte":
		return semanticRule{"canonical base64", CheckBase64}
	default:
		return semanticRule{}
	}
}

func installProtocolRules(compiler *jsonschema.Compiler, resources map[string]resource) {
	rules := make(map[string]semanticRule)
	for uri, entry := range resources {
		if !entry.protocol {
			continue
		}
		// This document already passed exact pin and syntax checks. Location
		// discovery visits schema positions only and cannot fail here.
		err := walkSchemas(entry.document, "", func(object map[string]any, pointer string) error {
			format, _ := object["format"].(string)
			rule := protocolRule(pointer, format)
			if rule.check != nil {
				rules[uri+"#"+pointer] = rule
			}
			return nil
		})
		if err != nil {
			panic(err) // visitor above cannot return an error
		}
	}
	compiler.RegisterVocabulary(&jsonschema.Vocabulary{
		URL: "https://schema.invalid/vocab/mcp-locations-v1",
		Compile: func(ctx *jsonschema.CompilerContext, object map[string]any) (jsonschema.SchemaExt, error) {
			location := ctx.Enqueue(nil).Location
			// References can target schema objects beneath unknown keywords.
			// Qualify every compiled node, including those outside the walk.
			if err := checkSchemaNode(object, location); err != nil {
				return nil, err
			}
			if rule, ok := rules[location]; ok {
				return rule, nil
			}
			return nil, nil
		},
	})
	compiler.AssertVocabs()
}
