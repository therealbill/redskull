package common

import "github.com/docker/libkv/store"

var (
	Backingstore store.Store
	StoreConfig  BackingStoreConfig
)

type BackingStoreConfig struct {
	ConstellationBase string
	SentinelBase      string
	Podbase           string
}
