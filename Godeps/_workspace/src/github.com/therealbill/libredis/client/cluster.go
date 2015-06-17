package client

import (
	"fmt"
	"log"

	"github.com/therealbill/libredis/structures"
)

func (cn *ClusterNode) HandleSlot(slot int) bool {
	return false
}

func (r *Redis) ClusterNodes() (nodes []structures.ClusterNode, err error) {
	return
}

func (r *Redis) ClusterSlots() (slots []structures.ClusterSlot, err error) {
	res, err := r.ExecuteCommand("CLUSTER", "SLOTS")
	if err != nil {
		log.Print(err)
		return
	}
	for i := 0; i < len(res.Multi); i++ {
		var masterh, slaveh string
		var masterp, slavep int64
		var slaves []string
		start, _ := res.Multi[i].Multi[0].IntegerValue()
		end, _ := res.Multi[i].Multi[1].IntegerValue()
		masterh, _ = res.Multi[i].Multi[2].Multi[0].StringValue()
		masterp, _ = res.Multi[i].Multi[2].Multi[1].IntegerValue()
		// need o handle the case of multiple slaves
		slaveh, _ = res.Multi[i].Multi[3].Multi[0].StringValue()
		slavep, _ = res.Multi[i].Multi[3].Multi[1].IntegerValue()
		slaves = append(slaves, fmt.Sprintf("%s:%d", slaveh, slavep))
		slot := structures.ClusterSlot{Start: start, End: end, MasterHost: masterh, MasterPort: masterp, Slaves: slaves}
		r.Slots = append(r.Slots, slot)
		_ = slavep
		_ = slaveh
	}
	return r.Slots, nil
}
