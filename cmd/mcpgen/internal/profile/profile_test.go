package profile

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

func reviewedInputs(t *testing.T) *Inputs {
	t.Helper()
	root, err := sdkmap.RuntimeModule(".")
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := Load(root, Revision)
	if err != nil {
		t.Fatalf("FAIL reviewed profile control: %v", err)
	}
	return inputs
}

func TestRepairA08InputPolicy(t *testing.T) {
	for name, mutate := range map[string]func(*Inputs){
		"missing-disposition": func(i *Inputs) { delete(i.Mapping.Dispositions, "Tool") },
		"unknown-disposition": func(i *Inputs) { i.Mapping.Dispositions["Tool"] = "skip" },
		"missing-reason":      func(i *Inputs) { i.Mapping.Reasons["Tool"] = "" },
		"missing-Go-name":     func(i *Inputs) { delete(i.Mapping.GoNames, "Tool") },
		"colliding-Go-name":   func(i *Inputs) { i.Mapping.GoNames["Tool"] = "Implementation" },
		"invalid-Go-name":     func(i *Inputs) { i.Mapping.GoNames["Tool"] = "type" },
		"missing-method":      func(i *Inputs) { i.Methods = i.Methods[1:] },
		"unbound-method":      func(i *Inputs) { i.Methods[0].Pointer = "#/$defs/Tool" },
		"unbound-definition": func(i *Inputs) {
			i.Methods[0].Definition, i.Methods[0].Pointer = "Tool", "#/$defs/Tool"
		},
		"duplicate-method":            func(i *Inputs) { i.Methods[0] = i.Methods[1] },
		"missing-normative-reference": func(i *Inputs) { i.Methods[0].NormativeReference = "" },
		"wrong-normative-revision": func(i *Inputs) {
			i.Methods[0].NormativeReference = "https://modelcontextprotocol.io/specification/old/"
		},
		"missing-SDK-symbol":          func(i *Inputs) { i.Methods[0].SDKSymbol = "" },
		"missing-fixture":             func(i *Inputs) { i.Methods[0].Fixture = "" },
		"missing-cancellation":        func(i *Inputs) { i.Methods[0].Cancellation = nil },
		"missing-retry-policy":        func(i *Inputs) { i.Methods[0].RetryPolicy = "" },
		"missing-retry-safety":        func(i *Inputs) { i.Methods[0].RetrySafe = nil },
		"wrong-completion":            func(i *Inputs) { i.Methods[0].Completion = "notification" },
		"wrong-cancellation-scope":    func(i *Inputs) { i.Methods[0].CancellationScope = "parent-context" },
		"unknown-capability":          func(i *Inputs) { i.Methods[0].Capability = "notAdvertised" },
		"unbound-capability":          func(i *Inputs) { i.Methods[0].CapabilityPointer = "#/$defs/Tool" },
		"missing-contract-source":     func(i *Inputs) { i.Methods[0].ContractSource = "" },
		"missing-qualification-owner": func(i *Inputs) { i.Methods[0].QualificationOwner = "" },
		"missing-conditions":          func(i *Inputs) { i.Methods[0].Conditions = "" },
		"missing-schema":              func(i *Inputs) { i.Schema = nil },
	} {
		t.Run(name, func(t *testing.T) {
			inputs := reviewedInputs(t)
			mutate(inputs)
			if err := inputs.validate(); err == nil {
				t.Fatalf("FAIL invalid reviewed input policy accepted: %s", name)
			}
		})
	}
	i := reviewedInputs(t)
	if len(i.Mapping.Dispositions) != 155 || len(i.Methods) != 21 || i.Mapping.GoNames["RequestId"] != "RequestID" || i.Mapping.Dispositions["CreateMessageRequest"] != "owned-codec" {
		t.Error("FAIL reviewed definition/method/name/codec inventory differs")
	}
}

func TestRepairA08SDKContext(t *testing.T) {
	for _, broken := range []string{"valid", "source", "missing-definition", "unsafe-alias", "codec-downgrade", "missing-method-SDK-symbol"} {
		t.Run(broken, func(t *testing.T) {
			i := reviewedInputs(t)
			actual := &sdkmap.Result{
				Revision: Revision, SDKModule: i.Mapping.Module, SDKVersion: i.Mapping.SDKVersion,
				SDKSum: i.Mapping.SDKSum, SourceSHA256: i.Mapping.SDKContext,
				Dispositions: make(map[string]string),
			}
			for name, value := range i.Mapping.Dispositions {
				actual.Dispositions[name] = value
			}
			for _, method := range i.Methods {
				actual.SDKExports = append(actual.SDKExports, strings.TrimPrefix(method.SDKSymbol, "mcp."))
			}
			switch broken {
			case "source":
				actual.SourceSHA256 = strings.Repeat("0", 64)
			case "missing-definition":
				delete(actual.Dispositions, "Tool")
			case "unsafe-alias":
				actual.Dispositions["Tool"] = "owned"
			case "codec-downgrade":
				i.Mapping.Dispositions["Root"] = "owned"
			case "missing-method-SDK-symbol":
				actual.SDKExports = nil
			}
			result, err := i.ApplySDK(actual)
			if broken == "valid" {
				if err != nil || result.Dispositions["CreateMessageRequest"] != "owned-codec" {
					t.Fatalf("FAIL SDK-context control: %v", err)
				}
			} else if err == nil {
				t.Fatalf("FAIL SDK context or unsafe mapping accepted: %s", broken)
			}
		})
	}
}
