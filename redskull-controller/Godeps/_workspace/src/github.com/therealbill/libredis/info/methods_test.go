package info

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

var (
	infostring_raw          string
	infostring_commandstats string
	infostring_keyspace     string
)

func init() {
	dat, err := ioutil.ReadFile("info-string.txt")
	if err != nil {
		log.Fatal("unable to read file: ", err)
	}
	infostring_raw = string(dat)

	dat, err = ioutil.ReadFile("commandstats.txt")
	if err != nil {
		log.Fatal("unable to read file: ", err)
	}
	infostring_commandstats = string(dat)

	dat, err = ioutil.ReadFile("info-string.txt")
	if err != nil {
		log.Fatal("unable to read file: ", err)
	}
	infostring_keyspace = string(dat)
}

func TestBuildMapFromInfoString(t *testing.T) {
	trimmed := strings.TrimSpace(infostring_raw)
	rmap := BuildMapFromInfoString(trimmed)
	if len(rmap["redis_version"]) == 0 {
		t.Error("Version wasn't parsed")
		t.Fail()
	}
}

func TestBuildInfoKeyspace(t *testing.T) {
	space := BuildInfoKeyspace(infostring_keyspace)
	if len(space.Databases) == 0 {
		t.Fail()
	}
}

func TestCommandStats(t *testing.T) {
	stats := CommandStats(infostring_commandstats)
	if len(stats.Stats) == 0 {
		t.Fail()
	}
	// The act of calling CommandStats will produce at least one call it info
	// So we ensure we have at least one call
	if stats.Stats["info"]["calls"] == 0 {
		t.Fail()
	}
}

func TestKeyspaceStats(t *testing.T) {
	stats := KeyspaceStats(infostring_keyspace)
	if len(stats.Databases) == 0 {
		fmt.Printf("%+v\n", stats)
		t.Error("didn't get expected Keyspace Stats structure.")
		t.Fail()
	}
}

func TestBuildAllInfoMap(t *testing.T) {
	alldata := BuildAllInfoMap(infostring_raw)
	if len(alldata["CPU"]["used_cpu_sys"]) == 0 {
		fmt.Printf("alldata.cpu.used_cpu_sys: %+v\n", alldata["CPU"]["used_cpu_sys"])
		t.Error("didn't parse cpu.used_cpu_sys")
		t.Fail()
	}
}

func TestGetAllInfo(t *testing.T) {
	all := GetAllInfo(infostring_raw)
	// Server Checks
	if all.Server.Arch_bits == 0 {
		t.Error("Didn't parse Server.Arch_bits")
		t.Fail()
	}
}

func TestInfo(t *testing.T) {
	all := GetAllInfo(infostring_raw)
	// Server Checks
	if all.Server.Arch_bits == 0 {
		t.Error("Didn't parse Server.Arch_bits")
		t.Fail()
	}
	if len(all.Server.Version) == 0 {
		t.Error("Didn't parse Server.Version")
		t.Fail()
	}
	// Replication Checks
	if len(all.Replication.Role) == 0 {
		t.Error("Failed to parse Replication.Role")
		t.Fail()
	}

	// Persistence
	if all.Persistence.AOFEnabled {
		t.Error("Tests assume default config, so this AOFEnabled shoudl be false")
		t.Fail()
	}
	// Stats
	if all.Stats.TotalCommandsProcessed == 0 {
		t.Error("Failed to parse Stats.TotalCommandsProcessed")
		t.Fail()
	}
	// Memory
	if len(all.Memory.UsedMemoryHuman) == 0 {
		t.Error("Failed to parse Memory.UsedMemoryHuman")
		t.Fail()
	}
	if all.Memory.UsedMemory == 0 {
		t.Error("Failed to parse Memory.UsedMemory")
		t.Fail()
	}
	// Keyspace
	if len(all.Keyspace.Databases) == 0 {
		t.Error("Failed to parse at least one DB from Keyspace")
		t.Fail()
	}
	if all.Commandstats.Stats["info"]["calls"] == 0 {
		t.Error("Failed to parse stats on info command from Commandstats")
		fmt.Printf("Commandstats:\t%+v\n", all.Commandstats)
		t.Fail()
	}

}

func TestUpperFirst(t *testing.T) {
	instring := "this"
	outsting := upperFirst(instring)
	if instring == outsting {
		t.Error("Failed to convert this to This")
		t.Fail()
	}
	if outsting != "This" {
		t.Error("Failed to convert this to This")
		t.Fail()
	}

	empty := upperFirst("")
	if empty != "" {
		t.Error("upperFirst on empty strign result sin non-empty string")
		t.Fail()
	}
}
