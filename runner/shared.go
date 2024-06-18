package runner

import (
	"github.com/dodopizza/jaeger-kusto/config"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"
)

func Serve(c *config.PluginConfig, store shared.StoragePlugin, logger hclog.Logger) error {
	return serveServer(c, store, logger)
}

func newGRPCServerWithTracer(handler *shared.GRPCHandler) *grpc.Server {
	server := grpc.NewServer()

	healthServer := health.NewServer()
	reflection.Register(server)
	handler.Register(server, healthServer)
	return server
}
