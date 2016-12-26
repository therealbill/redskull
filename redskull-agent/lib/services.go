package lib

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	consul "github.com/hashicorp/consul/api"
)

var (
	lock *consul.Lock
)

// DiscoverableService is a service that registers itself with consul
type DiscoverableService interface {
	RegisterService() error
	SetPort(int)
	AddCheck(consul.AgentServiceCheck)
	EnableMaintenance(string) error
	DisableMaintenance() error
}

// ConsulConfiguredService is an interface for creating services configured
// from Consul
type ConsulConfiguredService interface {
	GetBase() string
	SetBase(string)
	SetValue(string, string) error
	GetString(string, bool) (string, error)
	GetInteger(string, bool) (int, error)
	GetLocalAddress() string
}

// LocalService is an interface for Local build services
type LocalService interface {
	DiscoverableService
	ConsulConfiguredService
}

type DiscoveredServices interface {
	Refresh() error
	GetHostPort() (string, error)
}

type RemoteService struct {
	c             *consul.Client
	services      []*consul.ServiceEntry
	Name          string
	Tag           string
	Connected     bool
	ConsulAddress string
}

func NewRemoteService(caddr, service, tag string) *RemoteService {
	rs := RemoteService{Name: service, Tag: tag, ConsulAddress: caddr}
	return &rs
}

func (s *RemoteService) Connect(a string) error {
	cfg := consul.DefaultConfig()
	if a > "" {
		cfg.Address = a
	}
	var err error
	s.c, err = consul.NewClient(cfg)
	if err != nil {
		return err
	}
	s.Connected = true
	return nil
}

// discoverServices queries Consul for services data
func (s *RemoteService) discoverServices() error {
	var err error
	if !s.Connected {
		err = s.Connect(s.ConsulAddress)
		if err != nil {
			return err
		}
	}
	// retrieve list of services
	s.services, _, err = s.c.Health().Service(s.Name, s.Tag, true, &consul.QueryOptions{})
	return err
}

func (s *RemoteService) GetHostPort() (string, int, error) {
	rand.Seed(time.Now().UnixNano())
	err := s.discoverServices()
	if err != nil {
		return "", 0, err
	}
	selected := s.services[rand.Intn(len(s.services)-1)]
	if selected == nil {
		return "", 0, errors.New("No available services")
	}
	return selected.Service.Address, selected.Service.Port, nil
}

func (s *RemoteService) GetEntries() (entries []string, err error) {
	rand.Seed(time.Now().UnixNano())
	err = s.discoverServices()
	if err != nil {
		return
	}
	log.Printf("found %d %q services", len(s.services), s.Name)
	for _, e := range s.services {
		entries = append(entries, fmt.Sprintf("%s:%d", e.Node.Address, e.Service.Port))
	}
	return
}
