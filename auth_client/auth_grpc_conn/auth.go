package auth_grpc_conn

import (
	"auth_client/proto"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func New(addr string) (proto.AuthClient, error) {
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, time.Second*5)
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("can`t connect to grpc %s: %v", addr, err)
	}
	return proto.NewAuthClient(conn), nil
}
