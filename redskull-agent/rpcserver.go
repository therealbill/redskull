package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	consul "github.com/hashicorp/consul/api"
	lib "github.com/therealbill/redskull/redskull-agent/lib"
)

type RedAgentService struct {
	constellation *lib.Constellation
	mu            *sync.RWMutex
	Confbase      string
	Name          string
	CellName      string
	Address       string
	Port          int
	ConsulAddress string
	client        *consul.Client
	ServiceChecks consul.AgentServiceChecks
	Multi         bool
	ID            string
	RPC           *RPC
}

// SECTION: CONSUL
// Connect will connect the service to Consul
func (s *RedAgentService) ConsulConnect() error {
	var err error
	cc := consul.DefaultConfig()
	cc.Address = s.ConsulAddress
	s.client, err = consul.NewClient(cc)
	if err != nil {
		log.Print("unable to open consul")
		return err
	}
	log.Print("connected to consul")
	return nil
}

// GetLocalAddress return the local address of the consul agent
func (s *RedAgentService) GetLocalAddress() string {
	agent := s.client.Agent()
	self, _ := agent.Self()
	myaddr := self["Member"]["Addr"].(string)
	return myaddr
}

//SetPort is used to set the port for the service. Must be called before
//RegisterService.
func (s *RedAgentService) SetPort(p int) {
	s.Port = p
}

// GetBase returns the base URL for configurations
func (s *RedAgentService) GetBase() string {
	return s.Confbase
}

// AddCheck adds a consul.AgentServiceCheck
// will only be effective if issued prior to RegisterService
func (s *RedAgentService) AddCheck(c consul.AgentServiceCheck) {
	s.ServiceChecks = append(s.ServiceChecks, &c)
}

// SetBase sets the base URL for configurations
func (s *RedAgentService) SetBase(b string) {
	s.Confbase = b
}

// SetValue updates/sets the specified key and value in Consul
func (s *RedAgentService) SetValue(k, v string) error {
	//TODO implement SetValue, returning an error until implemented
	mkey := s.Confbase + k
	key := strings.TrimPrefix(mkey, "/")
	kp := consul.KVPair{Key: key, Value: []byte(v)}
	log.Printf("kp: %+v", kp)
	kv := s.client.KV()
	wm, err := kv.Put(&kp, nil)
	log.Printf("Put got '%+v'", wm)
	return err
}

func (s *RedAgentService) RegisterPod(name, ip string, port int) error {
	podbase := "pods/" + s.Address + "/" + name + "/"
	err := s.SetValue(podbase+"ip", ip)
	if err != nil {
		return err
	}
	err = s.SetValue(podbase+"port", ip)
	return err
}

// GetString returns the value in consul for the given key name
// if bail is set, it will caue a Fatal exit. This is usweful for
// specifying that the value must be found for the program to continue.
func (s *RedAgentService) GetString(k string, bail bool) (string, error) {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair, _, err := kv.Get(kurl, nil)
	if err != nil {
		if bail {
			log.Fatal(errors.New("unable to communicate with backing store"))
		}
		return "", err
	}
	if pair == nil {
		return "", errors.New("no such key")
	}
	return string(pair.Value), err
}

// GetInteger returns the value in consul for the given key name
// if bail is set, it will caue a Fatal exit. This is usweful for
// specifying that the value must be found for the program to continue.
func (s *RedAgentService) GetInteger(k string, bail bool) (int, error) {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair, _, err := kv.Get(kurl, nil)
	if err != nil {
		if bail {
			log.Fatal(errors.New("unable to communicate with backing store"))
		}
		return 0, err
	}
	if pair == nil {
		return 0, errors.New("key not found")
	}
	return strconv.Atoi(string(pair.Value))
}

// RegisterService registers the service in Consul
func (s *RedAgentService) RegisterService() error {
	if s.Port == 0 {
		return errors.New("Need to call SetPort(n) before calling RegisterService")
	}
	s.ID = fmt.Sprintf("%s", s.Name)
	agent := s.client.Agent()
	self, _ := agent.Self()
	myaddr := self["Member"]["Addr"].(string)
	asr := &consul.AgentServiceRegistration{
		ID:      s.ID,
		Name:    s.Name,
		Port:    s.Port,
		Checks:  s.ServiceChecks,
		Address: myaddr,
		Tags:    []string{s.CellName},
	}
	log.Printf("asr: %+v", asr)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		log.Printf("Deregistering service %s", s.ID)
		agent.ServiceDeregister(s.ID)
		log.Printf("Deregistered service %s", s.ID)
		os.Exit(0)
	}()
	return agent.ServiceRegister(asr)
}

// EnableMaintenance takes a string as the reason and switches on Consul's
// maintenance mode for the service. NOTE: This only works when a service
// is singular-per-host because we can't determine the ID of a multi-service.
// In order to make this work for multi's we would have to be able to specify
// the full service ID, which also means writing a lister for all matching
// services on the host. Frankly, given how rare that should be, I don't think
// it is worth the added effort.
func (s *RedAgentService) EnableMaintenance(r string) error {
	log.Printf("Enabling maintenance on %q", s.Name)
	s.ConsulConnect()
	return s.client.Agent().EnableServiceMaintenance(s.Name, r)
}

// DisableMaintenance takes a string as the reason and switches on Consul's maintenance mode for the service
func (s *RedAgentService) DisableMaintenance() error {
	s.ConsulConnect()
	log.Printf("Disabling maintenance on %q", s.Name)
	return s.client.Agent().DisableServiceMaintenance(s.Name)
}

// ServeRPC is used to start serving the RPC interface using config pulled from Consul
func (s *RedAgentService) ServeRPC() {
	rpc.Register(s.RPC)
	rpc_on := fmt.Sprintf("%s:%d", s.Address, s.Port)
	l, e := net.Listen("tcp", rpc_on)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	rpc.Accept(l)
}

func NewRedAgentService(cell string) RedAgentService {
	sc := RedAgentService{
		Confbase:      "/svcconfig/redskull-agent/" + cell + "/",
		Name:          "redskull-agent",
		Multi:         false,
		ConsulAddress: "localhost:8500",
		CellName:      cell,
	}
	log.Printf("sc: %+v", sc)
	sc.ConsulConnect()
	sc.Port, _ = sc.GetInteger("config/rpcport", true)
	sc.Address = sc.GetLocalAddress()
	conf, err := sc.GetString("config/sentinelconfig", true)
	if err != nil {
		log.Printf("Err: %v", err)
	}
	sc.RPC = NewRPC(sc.Address, conf, cell, sc.Address)
	pods, err := sc.RPC.GetPods()
	for n, c := range pods {
		log.Printf("Need to register pod %v (%+v)", n, c)
		err = sc.RegisterPod(n, c.IP, c.Port)
		if err != nil {
			log.Printf("unable to register pod. Error was %v", err)
		}
	}
	sc.RegisterService()
	return sc
}

// SECTION: SENTINEL

type Item struct {
	key  string
	Item interface{}
}

type RPC struct {
	constellation *lib.Constellation
	mu            *sync.RWMutex
}

func badContextError(err error) {
	if err != nil {
		panic("Constellation is not initialized")
	}
}

func (r *RPC) GetPodAuth(podname string, resp *string) (err error) {
	log.Printf("mpc: %+v", r.constellation.SentinelConfig.ManagedPodConfigs)
	pod, exists := r.constellation.SentinelConfig.ManagedPodConfigs[podname]
	if !exists {
		err = errors.New("Pod Not found")
		return err
	}
	*resp = pod.AuthToken
	return err
}

func (r *RPC) GetPods() (map[string]lib.SentinelPodConfig, error) {
	return r.constellation.SentinelConfig.ManagedPodConfigs, nil
}

func NewRPC(name, config, cell, addr string) *RPC {
	con, err := lib.GetConstellation(name, config, cell, addr)
	if err != nil {
		log.Fatal(err)
	}
	return &RPC{
		constellation: &con,
	}
}
