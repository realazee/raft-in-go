# Project 4: Raft

## Introduction

This is a Go implementation of Raft, a replicated state machine protocol. A replicated service achieves fault tolerance by storing complete copies of its state (i.e., data) on multiple replica servers. Replication allows the service to continue operating even if some of its servers experience failures (crashes or a broken or flaky network). The challenge is that failures may cause the replicas to hold differing copies of the data.

Raft organizes client requests into a sequence, called the log, and ensures that all the replica servers see the same log. Each replica executes client requests in log order, applying them to its local copy of the service's state. Since all the live replicas see the same log contents, they all execute the same requests in the same order, and thus continue to have identical service state. If a server fails but later recovers, Raft takes care of bringing its log up to date. Raft will continue to operate as long as at least a majority of the servers are alive and can talk to each other. If there is no such majority, Raft will make no progress, but will pick up where it left off as soon as a majority can communicate again.

Raft is implemented here as a Go object type with associated methods, meant to be used as a module in a larger service. A set of Raft instances talk to each other with RPC to maintain replicated logs. Your Raft interface will support an indefinite sequence of numbered commands, also called log entries. The entries are numbered with index numbers. The log entry with a given index will eventually be committed. At that point, Raft should send the log entry to the larger service for it to execute.

## Getting Started

To get up and running, execute the following commands. Don't forget the `git pull` to get the latest software.

```
$ cd raft
$ go test -race
Test (4A): initial election ...
--- FAIL: TestInitialElection4A (5.04s)
        config.go:326: expected one leader, got none
Test (4A): election after network failure ...
--- FAIL: TestReElection4A (5.03s)
        config.go:326: expected one leader, got none
...
```

To run a specific set of tests, use `go test -race -run 4A` or `go test -race -run TestInitialElection4A`.

## The Code

The implementation supports the following interface:

```
// create a new Raft server instance:
rf := Make(peers, me, persister, applyCh)

// start agreement on a new log entry:
rf.Start(command interface{}) (index, term, isleader)

// ask a Raft for its current term, and whether it thinks it is leader
rf.GetState() (term, isLeader)

// each time a new entry is committed to the log, each Raft peer
// should send an ApplyMsg to the service (or Tester).
type ApplyMsg
```

A service calls `Make(peers,me,…)` to create a Raft peer. The peers argument is an array of network identifiers of the Raft peers (including this one), for use with RPC. The `me` argument is the index of this peer in the peers array. `Start(command)` asks Raft to start the processing to append the command to the replicated log. `Start()` should return immediately, without waiting for the log appends to complete. The service expects your implementation to send an `ApplyMsg` for each newly committed log entry to the `applyCh` channel argument to `Make()`.

`raft.go` contains example code that sends an RPC (`sendRequestVote()`) and that handles an incoming RPC (`RequestVote()`). Raft peers will exchange RPCs using the labrpc Go package (source in `src/labrpc`). The Tester can tell `labrpc` to delay RPCs, re-order them, and discard them to simulate various network failures. 

```
$ for i in {0..10}; do go test; done
```
