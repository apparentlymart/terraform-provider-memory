package main

import (
	"log"

	"github.com/apparentlymart/terraform-provider-memory/internal/memory"
	"github.com/apparentlymart/terraform-provider-memory/internal/tfplugin6"
	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"
)

// protocolVersion1 is an implementation of rpcplugin.ServerVersion that implements
// protocol version 6.
type protocolVersion6 struct {
	logger *log.Logger
}

// protocolVersion1 must implement the rpcplugin.ServerVersion interface
var _ rpcplugin.ServerVersion = protocolVersion6{}

func (p protocolVersion6) RegisterServer(server *grpc.Server) error {
	tfplugin6.RegisterProviderServer(server, memory.NewProvider(p.logger))
	return nil
}
