package main

import (
	graceful "github.com/leaq-ru/lib-graceful"
	"github.com/leaq-ru/proto/codegen/go/user"
	"github.com/leaq-ru/user/config"
	"github.com/leaq-ru/user/logger"
	"github.com/leaq-ru/user/roleimpl"
	"github.com/leaq-ru/user/userimpl"
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
	user.RegisterRoleServer(srv, roleimpl.NewServer())

	lis, err := net.Listen("tcp", strings.Join([]string{
		"0.0.0.0",
		config.Env.Grpc.Port,
	}, ":"))
	logger.Must(err)

	logger.Must(srv.Serve(lis))
}
