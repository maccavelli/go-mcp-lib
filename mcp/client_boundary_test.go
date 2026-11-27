package mcp

import (
	"bytes"
	legacyjson "encoding/json"
	json "encoding/json/v2"
	"errors"
	"testing"

	"github.com/maccavelli/go-mcp-lib/mcp/internal/sdkbridge"
	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOwnedResultBoundary(t *testing.T) {
	for _, invalid := range []*sdk.CallToolResult{nil, {}} {
		value, err := convertClientResult[mcplib.CallToolResult]("fixture", invalid)
		if value != nil || !errors.Is(err, ErrToolCall) {
			t.Errorf("FAIL invalid owned result accepted or unclassified: %+v %v", value, err)
		}
	}
	var source sdk.CallToolResult
	if err := legacyjson.Unmarshal([]byte(`{"resultType":"complete","content":[{"type":"text","text":"preserved"}],"isError":false}`), &source); err != nil {
		t.Fatal(err)
	}
	source.StructuredContent = map[string]any{"exact": legacyjson.Number("9007199254740993")}
	owned, err := convertClientResult[mcplib.CallToolResult]("fixture", &source)
	if err != nil {
		t.Fatal(err)
	}
	text, err := owned.Content[0].AsTextContent()
	if err != nil || text.Text != "preserved" || owned.ResultType != "complete" {
		t.Fatalf("FAIL owned content or discriminator lost: %+v %v", owned, err)
	}
	body, err := json.Marshal(owned)
	if err != nil || !bytes.Contains(body, []byte("9007199254740993")) {
		t.Errorf("FAIL exact SDK numeric value lost: %s %v", body, err)
	}
	roundTrip, err := sdkbridge.ToSDK[sdk.CallToolResult](owned)
	if err != nil || len(roundTrip.Content) != 1 || roundTrip.NeedsInput() {
		t.Errorf("FAIL owned-to-SDK conversion: %+v %v", roundTrip, err)
	}
}
