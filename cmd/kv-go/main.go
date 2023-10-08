package main

import (
	"flag"
	"fmt"
	"github.com/Boar-D-White-Foundation/kvgo"
	"github.com/Boar-D-White-Foundation/kvgo/internal/rtimer"
	"log/slog"
	"strings"
	"time"
)

func main() {
	slog.Info("hello!")

	config, err := parseConfig()
	if err != nil {
		slog.Error("can't parse config", "error", err)
		return
	}

	raft := kvgo.MakeRaft(*config, rtimer.Default())
	rest := makeRestServer(config, &raft)
	raft.SetHandler(&rest)

	go func() {
		rest.StartHttpServer()
		raft.Stop()
	}()

	err = raft.Start()
	if err != nil {
		slog.Error("raft fatal error", "error", err)
		return
	}
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
		ElectionTimeoutMax: 30000 * time.Millisecond,
		ElectionTimeoutMin: 15000 * time.Millisecond,
		LeaderHeartbeat:    1000 * time.Millisecond,
		CurrentId:          currentId,
		Cluster:            cluster,
	}, nil
}
