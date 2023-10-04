# ~~awesome~~ kv-go

# Disclaimer
This is an educational project about how to develop scalable distributed systems. Just for fun. It is **not** intended for practical application. Use this project at your own risk.

## Distributed in-memory (at least for now) KV storage written in Go

## Project Overview

### Objective

To design and implement a distributed in-memory key-value storage system using the Go programming language. 
The system should be capable of handling high throughput, be fault-tolerant, and easily scalable. 

#### Key Features:
* High Throughput: Initially, the system will be designed to handle 10K requests per second (RPS).
* Storage Capacity: Each node in the system will start with the capability to store 1 million key-value pairs.
* Fault Tolerance: The system will incorporate redundancy mechanisms to ensure data availability even in the event of node failures. This will be achieved through techniques such as data replication across multiple nodes. In case a node goes down, the system will automatically reroute requests to a healthy node, ensuring uninterrupted service.
* Scalability: The architecture will be designed to easily add or remove nodes based on demand. This dynamic scalability ensures that as the system grows, it can accommodate more data and handle increased traffic without significant changes to the underlying infrastructure.

### Scope

1. In-Memory Key-Value Storage
2. Data Replication
3. Data Sharding
4. Fault Tolerance
5. Consistency Guarantees
6. API for CRUD operations
7. Monitoring and Logging

## System Architecture

### Components

1. **Node**: Basic unit of the storage system, responsible for storing in-memory data.
2. **Cluster Manager**: Manages the nodes in the system.
3. **Client Library**: Used for interacting with the storage system.
4. **Monitoring Service**: Monitors system health and performance.

### Data Flow

1. The client sends a request via the client library.
2. The client library communicates with the Cluster Manager to find the appropriate node.
3. The node performs the operation and returns the result to the client library.
4. The client library returns the result to the client.

## Key Features

### In-Memory Key-Value Storage

- **Keys**: Strings
- **Values**: Arbitrary binary data
- **Storage Engine**: In-memory only

### Data Replication

- **Strategy**: Leader-follower replication
- **Consistency**: Eventual consistency

### Data Sharding

- **Strategy**: Consistent Hashing
- **Rebalancing**: Automatic upon node failure or addition

### Fault Tolerance

- **Node Failures**: Automatic failover to replicas
- **Data Recovery**: From replicas

### API
- `Put(key, value)`: Store a key-value pair. This operation typically goes to the leader of the shard to ensure data consistency.
- `Get(key, consistency="strong")`: Retrieve the value associated with the given key. By default, this reads from the leader to ensure the most up-to-date data. If consistency="eventual", it might read from a follower, potentially returning stale data.
- `Delete(key)`: Remove the key-value pair associated with the given key. This operation typically goes to the leader for consistency.
- `Update(key, value)`: Modify the value associated with the given key. Like Put, this operation goes to the leader.

### Monitoring and Logging

- **Metrics**: Latency, Throughput, Error Rates
- **Logs**: System logs, Error logs

## Technology Stack

- **Language**: Go
- **Communication**: CLI+gRPC for internal
- **Data Serialization**: Protocol Buffers
- **Monitoring**: Prometheus and Grafana

## Development Plan

TBD

## Risks and Mitigations

- **Data Loss**: Mitigated by replication. However, since this is an in-memory system, data will be lost if all replicas of a shard are lost.
- **Node Failures**: Mitigated by automatic failover and recovery.
- **Inconsistent Data**: Mitigated by eventual consistency model.

## Future Enhancements

* Strong consistency options
* Multi-region support
* Time-To-Live (TTL) for keys
* Scale throughput up to 100K RPS
* Leaderless approach

## Conclusion

This design document outlines the architecture and key features of a distributed in-memory key-value storage system. The project will be implemented in Go and aims to provide a high-throughput, fault-tolerant, and scalable storage solution.