package mcp

import (
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Transport is the owned seam for attaching a connection. The
// framework's guarded transports (stage 7) implement it; the SDK
// transport stays behind it.
type Transport interface {
	// transport returns the underlying SDK transport. Framework
	// implementations wrap guarded I/O around it; this method is the
	// single crossing point.
	transport() sdk.Transport
}

// StdioTransport connects over the process's stdin/stdout.
type StdioTransport struct{}

func (StdioTransport) transport() sdk.Transport {
	return &sdk.StdioTransport{}
}

// InMemoryTransport creates a connected pair for testing: the returned
// transports are the two ends of one pipe.
func InMemoryTransport() (Transport, Transport) {
	client, server := sdk.NewInMemoryTransports()
	return inMem{client}, inMem{server}
}

type inMem struct{ sdk sdk.Transport }

func (t inMem) transport() sdk.Transport { return t.sdk }
