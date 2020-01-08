package server

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func newPlugin(name, addr string) (*plugin, error) {
	fmt.Printf("making a connection to remote server\n")
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Printf("did not connect: %v\n", err)
		return nil, errors.Wrapf(err, "Failed to make connection")
	}

	return &plugin{
		name: name,
		addr: addr,
		conn: conn,
	}, nil
}
