package lib

import "github.com/therealbill/libredis/structures"

// NewPod will return a RedisPod construct. It requires the nae, address, port,
// and authentication token.
func NewPod(name, address string, port int, auth string) (rp RedisPod, err error) {
	rp.Name = name
	rp.AuthToken = auth
	return

}

// NewMasterFromMasterInfo accepts a MasterInfo struct from libredis/client
// combined with an authentication token to use and returns a RedisPod
// instance.
func NewMasterFromMasterInfo(mi structures.MasterInfo, authtoken string) (rp RedisPod, err error) {
	rp.Name = mi.Name
	rp.Info = mi
	rp.AuthToken = authtoken
	return rp, nil
}
