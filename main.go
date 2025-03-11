package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.rpcplugin.org/rpcplugin"
	"go.rpcplugin.org/rpcplugin/plugintrace"
)

func main() {
	logger := log.New(os.Stderr, "server: ", log.Flags())
	ctx := plugintrace.WithServerTracer(context.Background(), plugintrace.ServerLogTracer(logger))

	err := rpcplugin.Serve(ctx, &rpcplugin.ServerConfig{
		Handshake: rpcplugin.HandshakeConfig{
			// The client and server must both agree on the CookieKey and
			// CookieValue so that the server can detect whether it's running
			// as a child process of its expected client. If not, it will
			// produce an error message an exit immediately.
			CookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
			CookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
		},
		ProtoVersions: map[int]rpcplugin.ServerVersion{
			6: protocolVersion6{
				logger: logger,
			},
		},
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
