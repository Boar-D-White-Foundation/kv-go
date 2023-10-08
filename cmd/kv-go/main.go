package main

import (
	"flag"
	"fmt"
	"github.com/Boar-D-White-Foundation/kvgo"
	"log/slog"
	"strings"
)

func main() {
	slog.Info("hello!")

	config, err := parseConfig()
	if err != nil {
		slog.Error("can't parse config", "error", err)
		return
	}
	raft := MakeRaftService(*config)

	go func() {
		raft.StartHttpServer()
		raft.Stop()
	}()

	raft.Start()
	slog.Info("server stopped")
}

func parseConfig() (*kvgo.Config, error) {
	portParam := flag.String("port", "", "current port")
	clusterParam := flag.String("cluster", "", "all ports in cluster")

	flag.Parse()
	if *portParam == "" {
		return nil, fmt.Errorf("you need to pass port: --port=33001")
	}
	if *clusterParam == "" {
		return nil, fmt.Errorf("you need to pass cluster: --cluster=33001,33002,33003")
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
		return nil, fmt.Errorf("not found current port in cluster")
	}

	return &kvgo.Config{
		ElectionTimeoutMax: 30000,
		ElectionTimeoutMin: 15000,
		LeaderHeartbeat:    1000,
		CurrentId:          currentId,
		Cluster:            cluster,
	}, nil
}
