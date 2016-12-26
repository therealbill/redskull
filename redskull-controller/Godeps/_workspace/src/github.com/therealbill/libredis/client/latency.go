package client

import (
	"fmt"
	"log"
	//"reflect"
	//"strconv"
)

// LatencyRecord records a laency entry
type LatencyRecord struct {
	Timestamp    int64
	Milliseconds int64
}

// LatencyHistory contains the available LatencyRecords for a given event
type LatencyHistory struct {
	Event   string
	Records []LatencyRecord
}

//SetLatencyThreshold sets the number of milliseconds an event must reach to trigger storage
func (r *Redis) SetLatencyThreshold(ms int) error {
	res := r.ConfigSetInt("latency-monitor-threshold", ms)
	return res
}

//LatencyResetAll reselts counters for all events
func (r *Redis) LatencyResetAll() error {
	_, err := r.ExecuteCommand("latency", "reset")
	return err
}

//LatencyResetEvent reselts counters for all the specified event
func (r *Redis) LatencyResetEvent(event string) error {
	_, err := r.ExecuteCommand("latency", "reset", event)
	return err
}

//LatencyResetEvents reselts counters for all the specified events
func (r *Redis) LatencyResetEvents(events []string) error {
	for _, s := range events {
		_, _ = r.ExecuteCommand("latency", "reset", s)
	}
	return nil
}

//LatencyHistory returns the available LatencyRecords for an event
func (r *Redis) LatencyHistory(event string) (history LatencyHistory, err error) {
	history = LatencyHistory{Event: event}
	res, err := r.ExecuteCommand("latency", "history", event)
	if err != nil {
		log.Print(err)
		return history, err
	}
	for _, val := range res.Multi {
		var record LatencyRecord
		for i, entry := range val.Multi {
			switch i {
			case 0:
				record.Timestamp, _ = entry.IntegerValue()
			case 1:
				record.Milliseconds, _ = entry.IntegerValue()
			}
		}
		history.Records = append(history.Records, record)
	}
	return history, nil
}

// LatencyDoctor returns the outut form the Redis latency doctor command
func (r *Redis) LatencyDoctor() (text string, err error) {
	res, err := r.ExecuteCommand("latency", "doctor")
	if err != nil {
		return text, fmt.Errorf("Latency not supported on this instance")
	}
	return res.StringValue()
}
