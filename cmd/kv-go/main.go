package main

import (
	"flag"
	"fmt"
	kv_go "github.com/Boar-D-White-Foundation/kv-go"
	"log"
	"strings"
)

func main() {
	log.Println("hello!")

	config := parseConfig()
	raft := MakeRaftService(config)

	go func() {
		raft.StartHttpServer()
		raft.Stop()
	}()

	raft.Start()
	log.Println("server stopped")
}

func parseConfig() kv_go.Config {
	portParam := flag.String("port", "", "current port")
	clusterParam := flag.String("cluster", "", "all ports in cluster")

	flag.Parse()
	if *portParam == "" {
		log.Fatalf("you need to pass port: --port=33001")
	}
	if *clusterParam == "" {
		log.Fatalf("you need to pass cluster: --cluster=33001,33002,33003")
	}

	currentId := -1
	cluster := make(map[int]string)
	ports := strings.Split(*clusterParam, ",")

	for i := 0; i < len(ports); i++ {
		cluster[i] = fmt.Sprintf("localhost:%s", ports[i])
		if ports[i] == *portParam {
			currentId = i
		}
	}

	if currentId < 0 {
		log.Fatalf("not found current port in cluster")
	}

	return kv_go.Config{
		ElectionTimeoutMax: 30000,
		ElectionTimeoutMin: 15000,
		LeaderHeartbeat:    1000,
		CurrentId:          currentId,
		Cluster:            cluster,
	}
}
