package main

import (
	graceful "github.com/nnqq/scr-lib-graceful"
	"github.com/nnqq/scr-proto/codegen/go/user"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	"github.com/nnqq/scr-user/userimpl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"strings"
)

func main() {
	srv := grpc.NewServer()
	go graceful.HandleSignals(srv.GracefulStop)

	grpc_health_v1.RegisterHealthServer(srv, health.NewServer())
	user.RegisterUserServer(srv, userimpl.NewServer())

	lis, err := net.Listen("tcp", strings.Join([]string{
		"0.0.0.0",
		config.Env.Grpc.Port,
	}, ":"))
	logger.Must(err)

	logger.Must(srv.Serve(lis))
}
