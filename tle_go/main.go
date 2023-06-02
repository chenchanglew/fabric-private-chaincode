package main

import (
	"fmt"
)

func main() {
	fmt.Println("--- in TLE_go main.go start to create grpc server.---")
	tlestate := &Tlestate{}
	// serve block listener.
	go ServePeer(tlestate)

	// serve metadata service.
	ServeMeta("127.0.0.1:50051", tlestate)
}
