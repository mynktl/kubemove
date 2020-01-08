package client

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Initializer func(*grpc.Server)

var mux sync.Mutex
var conn *grpc.ClientConn

// TODO args fn to get user input
func NewClient(name string, opt Initializer) error {
	sAddr, err := getServerAddr()
	if err != nil {
		fmt.Printf("Failed to get server details %v\n", err)
		return errors.Wrapf(err, "Insufficient server details")
	}

	cAddr, err := getClientAddr()
	if err != nil {
		fmt.Printf("Failed to get client details %v\n", err)
		return errors.Wrapf(err, "Insufficient client details")
	}

	mux.Lock()
	defer mux.Unlock()
	go createServer(cAddr, opt)
	if err != nil {
		fmt.Printf("Failed to create server\n")
		return errors.Wrapf(err, "Failed to create server")
	}

	conn, err = register(name, cAddr, sAddr)
	if err != nil {
		fmt.Printf("Failed to register plugin.. %v\n", err)
		return errors.Wrapf(err, "Failed to register plugin")
	}

	return nil
}

func createServer(cAddr *addr, opt Initializer) {
	defer shutdown()

	lis, err := net.Listen("tcp", cAddr.addr)
	if err != nil {
		fmt.Printf("Failed to create a server. %v\n", err)
		return
	}

	s := grpc.NewServer()
	opt(s)

	if err := s.Serve(lis); err != nil {
		fmt.Printf("Failed to server.. %v\n", err)
		return
	}
	fmt.Printf("for\n")
	return
}

func shutdown() {
	mux.Lock()
	fmt.Printf("shutting down the process\n")
	mux.Unlock()
	if conn == nil {
		fmt.Printf("This is an error.. connection is nil!!\n")
	} else {
		conn.Close()
	}
	os.Exit(1)
}
