package main

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Boar-D-White-Foundation/kvgo"
	"github.com/Boar-D-White-Foundation/kvgo/internal/rtimer"
)

func main() {
	slog.Info("hello!")
	/*
			Specify the port we want to use to listen for client requests using:
		lis, err := net.Listen(...).
		Create an instance of the gRPC server using grpc.NewServer(...).
		Register our service implementation with the gRPC server.
		Call Serve() on the server with our port details to do a blocking wait until the process is killed or Stop() is called.

	*/
	config, err := parseConfig()

	if err != nil {
		slog.Error("can't parse config", err)
		return
	}

	raft := kvgo.MakeRaft(*config, rtimer.Default())
	//rest := makeRestServer(config, &raft)
	grpc := makeGrpcServer(config, &raft)
	raft.SetHandler(grpc)

	go func() {
		grpc.Start()
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
