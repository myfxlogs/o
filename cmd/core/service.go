package main

import (
	"context"
	"net/http"

	connect "connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	dashboardv1 "phase/internal/gen/dashboardv1"
	"phase/internal/gen/dashboardv1/dashboardv1connect"
)

type coreService struct{}

func (coreService) Ping(context.Context, *dashboardv1.PingRequest) (*dashboardv1.PingResponse, error) {
	return &dashboardv1.PingResponse{Status: "ok"}, nil
}

// mountCore 挂 ConnectRPC 服务到 mux（connect-go v2：Server 注册 + connecthttp.Mount）。
func mountCore(mux *http.ServeMux) {
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, coreService{})
	connecthttp.Mount(mux, server)
}
