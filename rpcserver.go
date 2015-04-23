package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/therealbill/redskull/actions"
	"github.com/therealbill/redskull/handlers"
)

type NewPodRequest struct {
	Name   string
	IP     string
	Port   int
	Quorum int
	Auth   string
}

type Item struct {
	key  string
	Item interface{}
}

type RPC struct {
	constellation *actions.Constellation
	mu            *sync.RWMutex
}

func badContextError(err error) {
	if err != nil {
		panic("Constellation is not initialized")
	}
}

func (r *RPC) GetSentinelsForPod(podname string, resp *[]string) error {
	sentinels := r.constellation.GetSentinelsForPod(podname)
	var snames []string
	for _, s := range sentinels {
		snames = append(snames, s.Name)
	}
	*resp = snames
	return nil
}

func (r *RPC) AddPod(pr NewPodRequest, resp *actions.RedisPod) (err error) {
	gob.Register(actions.RedisPod{})
	ok, err := r.constellation.MonitorPod(pr.Name, pr.IP, pr.Port, pr.Quorum, pr.Auth)
	if err != nil {
		log.Printf("MonitorPod call ('%+v') Failed. Error: %s", pr, err.Error())
		return err
	}
	if !ok {
		log.Printf("MonitorPod call ('%+v') Failed. No Error", pr)
		err = errors.New("MonitorPod call returned false, no error")
	}
	time.Sleep(time.Second * 2)
	pod, err := r.constellation.GetPod(pr.Name)
	if pod == nil || pod.Name == "" {
		err = errors.New("New Pod Not found")
		return err
	}
	*resp = *pod
	return err

}
func (r *RPC) GetPod(podname string, resp *actions.RedisPod) (err error) {
	gob.Register(actions.RedisPod{})
	pod, err := r.constellation.GetPod(podname)
	if pod == nil || pod.Name == "" {
		err = errors.New("Pod Not found")
		return err
	}
	*resp = *pod
	return err
}

func (r *RPC) RemovePod(podname string, resp *bool) (err error) {
	ok, err := r.constellation.RemovePod(podname)
	*resp = ok
	return err
}

func (r *RPC) AddSentinel(address string, resp *bool) (err error) {
	err = r.constellation.AddSentinelByAddress(address)
	if err == nil {
		*resp = true
	}
	return err
}

func (r *RPC) BalancePod(podname string, resp *bool) (err error) {
	pod, err := r.constellation.GetPod(podname)
	if pod == nil || pod.Name == "" {
		err = errors.New("Pod Not found")
		*resp = false
	}
	r.constellation.BalancePod(pod)
	*resp = true
	return err
}

func NewRPC() *RPC {
	context, err := handlers.NewPageContext()
	badContextError(err)
	return &RPC{
		constellation: context.Constellation,
	}
}

func ServeRPC() {
	rpc.Register(NewRPC())
	rpc_on := fmt.Sprintf("%s:%d", config.BindAddress, config.RPCPort)
	l, e := net.Listen("tcp", rpc_on)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	rpc.Accept(l)
}
