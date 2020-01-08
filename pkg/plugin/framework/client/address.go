package client

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

const (
	DEFAULT_CLIENT_PORT = "8000"
)

type addr struct {
	addr string
	cert string
}

const (
	ENV_SERVER_ADDR = "SERVER"
	ENV_SERVER_PORT = "SERVER_PORT"
	ENV_CLIENT_ADDR = "CLIENT"
	ENV_CLIENT_PORT = "CLIENT_PORT"
	ENV_SERVER_CERT = "CERT"
)

func getClientAddr() (*addr, error) {
	var err error

	caddr := os.Getenv(ENV_CLIENT_ADDR)
	cport := os.Getenv(ENV_CLIENT_PORT)

	fmt.Println(cport)

	if len(cport) == 0 {
		cport = DEFAULT_CLIENT_PORT
	}

	_, err = strconv.ParseUint(cport, 10, 16)
	if err != nil {
		fmt.Printf("Unable to parse port.. %v\n", err)
		return nil, errors.Wrapf(err, "Unable to parse port")
	}

	return &addr{
		addr: caddr + ":" + cport,
	}, nil
}

func getServerAddr() (*addr, error) {
	caddr := os.Getenv(ENV_SERVER_ADDR)
	//TODO
	cport := os.Getenv(ENV_SERVER_PORT)
	cport = "9000"
	cert := os.Getenv(ENV_SERVER_CERT)

	if len(cport) == 0 {
		fmt.Printf("Empty server address\n")
		return nil, errors.New("Insufficient server details")
	}

	_, err := strconv.ParseUint(cport, 10, 16)
	if err != nil {
		fmt.Printf("Unable to parse port.. %v\n", err)
		return nil, errors.Wrapf(err, "Unable to parse port")
	}

	return &addr{
		addr: caddr + ":" + cport,
		cert: cert,
	}, nil
}
