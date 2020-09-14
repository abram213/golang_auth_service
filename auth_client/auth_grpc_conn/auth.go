package auth_grpc_conn

import (
	"auth/proto"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func New(port string) (proto.AuthClient, error) {
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, time.Second*5)
	conn, err := grpc.DialContext(ctx, "127.0.0.1:"+port, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("cant connect to grpc: %v", err)
	}

	return proto.NewAuthClient(conn), nil
}
