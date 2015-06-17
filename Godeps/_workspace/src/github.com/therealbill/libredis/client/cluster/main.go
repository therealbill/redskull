package main

import (
	"log"

	"github.com/therealbill/libredis/client"
)

func main() {
	rc, _ := client.Dial("127.0.0.1", 30001)
	slots, err := rc.ClusterSlots()
	if err != nil {
		log.Print(err)
	}
	log.Printf("Slots: %+v", slots)
}
