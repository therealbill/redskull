// The actions package contains the code for interacting directly with Redis
// instances and taking actions against them. This includes higher level actions
// which apply to componets and to lower level actions which are taken against
// components directly.
package actions

import (
	"github.com/therealbill/libredis/structures"
	"github.com/therealbill/redskull/redskull-controller/common"
)

/*
// common.RedisPod is the construct used for holding data about a Redis Pod and taking
// action against it.
type common.RedisPod struct {
	Name                  string
	Info                  structures.MasterInfo
	Slaves                []structures.InfoSlaves
	Master                *RedisNode
	SentinelCount         int
	ActiveSentinelCount   int
	ReportedSentinelCount int
	AuthToken             string
	ValidAuth             bool
	ValidMasterConnection bool
	NeededSentinels       int
	MissingSentinels      bool
	TooManySentinels      bool
	HasInfo               bool
	NeedsReset            bool
	HasValidSlaves        bool
}
*/
// NewPod will return a common.RedisPod construct. It requires the nae, address, port,
// and authentication token.
func NewPod(name, address string, port int, auth string) (rp common.RedisPod, err error) {
	rp.Name = name
	rp.AuthToken = auth
	return

}

// NewMasterFromMasterInfo accepts a MasterInfo struct from libredis/client
// combined with an authentication token to use and returns a common.RedisPod
// instance.
func NewMasterFromMasterInfo(mi structures.MasterInfo, authtoken string) (rp common.RedisPod, err error) {
	rp.Name = mi.Name
	rp.Info = mi
	rp.AuthToken = authtoken
	return rp, nil
}
