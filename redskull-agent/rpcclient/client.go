package rsclient

import (
	"log"
	"net"
	"net/rpc"
	"time"
)

type Client struct {
	connection *rpc.Client
}

// NewPodRequest is a struct used for passing in the pod information from the
// client
type NewPodRequest struct {
	Name   string
	IP     string
	Port   int
	Quorum int
	Auth   string
}

// NewClient returns a client connection
func NewClient(dsn string, timeout time.Duration) (*Client, error) {
	connection, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{connection: rpc.NewClient(connection)}, nil
}

// GetPodAuth(podname) will return the lib.RedisPod type for the given pod, if
// found.
func (c *Client) GetPodAuth(podname string) (string, error) {
	var token string
	err := c.connection.Call("RPC.GetPodAuth", podname, &token)
	if err != nil {
		log.Print(err)
	}
	return token, err
}
