package raft

//
// This is an outline of the API that raft must expose to
// the service (or tester). See comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   Create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   Start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   Each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester) in the same server.
//
/*
Tests Presently Passing at least 50% of the time:
-All 4A
-Basic Agree
-RPC Byte
-FailAgree
-FailNoAgree
-Concurrent Start
-Count
-Unreliable Agree
-Persist 1
-Reliable Churn (10/10 first run, 9/10 on 2nd)
-Unreliable Churn
-Figure 8
-Figure 8 Unreliable
-Rejoin
-Persist 2


Tests Presently Failing more than 50% of the time:
-Backup (2/10 first trial, 6/10 the next, 4/10 after, 7/10 now)
-Persist 3



*/
import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"cs350/labgob"
	"cs350/labrpc"
)

// As each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). Set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

type LogEntry struct {
	Term    int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // This peer's index into peers[]
	dead      int32               // Set by Kill()
	//term      int
	votedFor            int // CandidateId that received vote in current term (or null if none)
	currentTerm         int
	log                 []LogEntry
	heartBeatTimeout    int //this is the election timeout.
	timeOfLastHeartBeat int
	isInElection        bool
	isLeader            bool
	votesForMe          int

	//4B
	commitIndex int
	lastApplied int
	nextIndex   []int
	matchIndex  []int
	applyCh     chan ApplyMsg

	// Your data here (4A, 4B).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

}

// Return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// Your code here (4A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.isLeader
	//// fmt.Println("GetState called, leader is ", isleader, " at term ", term, " by ", rf.me)
	return term, isleader
}

// Save Raft's persistent state to stable storage, where it
// can later be retrieved after a crash and restart. See paper's
// Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	//fmt.Println("persistingdata for ", rf.me, " at term ", rf.currentTerm, "with log", rf.log)
	//if len(rf.log) == 1 {
	//fmt.Println("WE'RE PERSISTING AN EMPTY LOG for ", rf.me)
	//}

	// Your code here (4B).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	// if rf.isLeader {
	// 	e.Encode(rf.nextIndex)
	// 	e.Encode(rf.matchIndex)
	// }

	data := w.Bytes()
	rf.persister.SaveRaftState(data)

}

// Restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		fmt.Println("readpersist exiting early on line 140")
		return
	}
	// Your code here (4B).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var term int
	var votedFor int
	var log []LogEntry
	// var nextIndex []int
	// var matchIndex []int
	if d.Decode(&term) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil {
		//fmt.Println("readpersist error for always persistent data")
	} else {
		rf.currentTerm = term
		rf.votedFor = votedFor
		rf.log = log
		//fmt.Println("Persist found log for ", rf.me, " at term ", rf.currentTerm, "with log", rf.log)
		//rf.persist()
	}
	// if d.Decode(&rf.nextIndex) != nil ||
	// 	d.Decode(&rf.matchIndex) != nil {
	// 	// fmt.Println("readpersist error for leader data")
	// } else {
	// 	rf.nextIndex = nextIndex
	// 	rf.matchIndex = matchIndex
	// }
	//

	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// Example RequestVote RPC arguments structure.
// Field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (4A, 4B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// Example RequestVote RPC reply structure.
// Field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (4A).
	Term        int
	VoteGranted bool
}

// Example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (4A, 4B).
	//begin to have servers vote for each other
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//rf.persist()
	// fmt.Println("RequestVote recieved by", rf.me, "at term", rf.currentTerm, "from", args.CandidateId, "voted for", rf.votedFor, "my log is ", rf.log)
	if args.Term <= rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		// fmt.Println("RequestVote denied due to term by", rf.me, " args term", args.Term, "current term", rf.currentTerm, "for candidate ", args.CandidateId)
		return
	}
	// fmt.Println("Candidate is", args.CandidateId, "Candidate last log index: ", args.LastLogIndex, " and my last log index: ", len(rf.log)-1)
	// fmt.Println("Candidate last log term: ", args.LastLogTerm, " and my last log term: ", rf.log[len(rf.log)-1].Term)
	//If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote
	if (args.LastLogTerm > rf.log[len(rf.log)-1].Term) || (args.LastLogTerm == rf.log[len(rf.log)-1].Term && args.LastLogIndex >= len(rf.log)) {
		rf.votedFor = args.CandidateId
		rf.currentTerm = args.Term
		reply.Term = rf.currentTerm
		// fmt.Println("RequestVote granted by", rf.me, "at term", rf.currentTerm, "for candidate", args.CandidateId, "term of candidate", args.Term)
		reply.VoteGranted = true
		//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
		rf.persist()

		return
	}
	rf.currentTerm = args.Term
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
	//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
	rf.persist()
	// fmt.Println("RequestVote denied due to votedfor nonmatch by", rf.me, "args term", args.Term, "current term", rf.currentTerm, "for candidate ", args.CandidateId)
}

// Example code to send a RequestVote RPC to a server.
// Server is the index of the target server in rf.peers[].
// Expects RPC arguments in args. Fills in *reply with RPC reply,
// so caller should pass &reply.
//
// The types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// Look at the comments in ../labrpc/labrpc.go for more details.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// appendEntries RPC
type AppendEntriesArgs struct {
	// 4A, my own func
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}
type AppendEntriesReply struct {
	// 4A, my own func
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// 4A, my own func
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
	rf.heartBeatTimeout = rand.Intn(700) + 800
	//// fmt.Println(args)
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		rf.isInElection = false
		rf.isLeader = false
		rf.votedFor = -1
		rf.votesForMe = 0
		//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
		rf.persist()
		// fmt.Println("AppendEntries denied due to term by", rf.me, " args term", args.Term, "current term", rf.currentTerm, "for leader ", args.LeaderId)
		return
	}
	// fmt.Println("In AppendEntries, prevlogindex is ", args.PrevLogIndex, " and prevlogterm is ", args.PrevLogTerm, "and entry count is", len(args.Entries), "recieved by:", rf.me, "sent by: ", args.LeaderId, "with entries ", args.Entries, "sender term", args.Term, "reciever term", rf.currentTerm)
	// fmt.Println("log is ", rf.log, "for ", rf.me)
	if log := rf.log; len(log) <= args.PrevLogIndex || log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		rf.isInElection = false
		rf.isLeader = false
		rf.votedFor = -1
		rf.votesForMe = 0
		//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
		// fmt.Println("AppendEntries denied due to log mismatch by", rf.me, " args term", args.Term, "current term", rf.currentTerm, "for leader ", args.LeaderId)
		rf.persist()
		return
	}
	//If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it
	//// fmt.Println("prevlogindex is ", args.PrevLogIndex, " and len of log is ", len(rf.log), "and entries are ", args.Entries)
	if log := rf.log; len(args.Entries) > 0 && len(log) > args.PrevLogIndex+1 && log[args.PrevLogIndex+1] != args.Entries[0] {
		rf.log = rf.log[:args.PrevLogIndex+1]

	}

	rf.log = append(rf.log[:args.PrevLogIndex+1], args.Entries...)

	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, len(rf.log)-1)
	}

	//// fmt.Println("AppendEntries recieved by", rf.me, "at term", rf.currentTerm, "from", args.LeaderId)
	//// fmt.Println("recieved by ", rf.me, "The log and entries are ", rf.log, "and ", args.Entries)
	rf.currentTerm = args.Term
	rf.isInElection = false
	rf.isLeader = false
	rf.votedFor = -1
	rf.votesForMe = 0
	reply.Term = rf.currentTerm
	reply.Success = true
	//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
	rf.persist()
	//// fmt.Println("AppendEntries accepted by", rf.me, "at term", rf.currentTerm, "from", args.LeaderId)
	//rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
	// rf.heartBeatTimeout = rand.Intn(700) + 800

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// The service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. If this
// server isn't the leader, returns false. Otherwise start the
// agreement and return immediately. There is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. Even if the Raft instance has been killed,
// this function should return gracefully.
//
// The first return value is the index that the command will appear at
// if it's ever committed. The second return value is the current
// term. The third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := 1
	term := 0
	// Your code here (4B).
	//// fmt.Println("Start called by ", rf.me, " at term ", rf.currentTerm)
	if rf.isLeader {

		rf.log = append(rf.log, LogEntry{rf.currentTerm, command})
		index = len(rf.log) - 1
		term = rf.currentTerm
		//send appendentries here

		for peer := range rf.peers {
			if peer != rf.me && rf.isLeader {
				//// fmt.Printf("address of nextIndex %p\n", &rf.nextIndex[peer])
				// // fmt.Println("peer is ", peer, "me is ", rf.me)
				args := AppendEntriesArgs{}
				args.Entries = make([]LogEntry, 0)
				args.Entries = append(args.Entries, rf.log[rf.nextIndex[peer]:]...)
				args.Term = rf.currentTerm
				args.LeaderId = rf.me
				args.PrevLogIndex = rf.nextIndex[peer] - 1
				args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
				args.LeaderCommit = rf.commitIndex
				go func(i int, peerIndex int, logLength int) {
					reply := AppendEntriesReply{}
					ok := rf.sendAppendEntries(i, &args, &reply)
					if !ok {
						return
					}
					if reply.Success {
						rf.mu.Lock()
						// // fmt.Println(len(log))
						rf.nextIndex[i] = logLength
						rf.matchIndex[i] = logLength - 1
						rf.mu.Unlock()
					} else {
						//// fmt.Println("AppendEntries failed for ", i, " at term ", rf.currentTerm, " by ", rf.me)
						rf.mu.Lock()
						rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
						if reply.Term > rf.currentTerm {
							// fmt.Println("term mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
							rf.currentTerm = reply.Term
							rf.isInElection = false
							rf.isLeader = false
							rf.votedFor = -1
							rf.votesForMe = 0
							//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
							rf.persist()
							//add else if for incorrect term, just return do nothing
						} else if reply.Term != rf.currentTerm {
							rf.mu.Unlock()
							return
						} else {
							// handle prevLog mismatch
							// fmt.Println("prevLog mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
							if peerIndex > 1 {
								rf.nextIndex[i] = peerIndex - 1
							}
						}
						rf.mu.Unlock()
					}

				}(peer, rf.nextIndex[peer], len(rf.log))
			}
		}

		// fmt.Println("Start called by ", rf.me, " at term ", rf.currentTerm)
		rf.persist()
		return index, term, true
	}
	rf.persist()
	return index, term, false
}

// The tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. Your code can use killed() to
// check whether Kill() has been called. The use of atomic avoids the
// need for a lock.
//
// The issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. Any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	//// fmt.Println("killed node", rf.me, "at term", rf.currentTerm)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {
		//polling interval
		//// fmt.Println("ticker looped by ", rf.me, " at term ", rf.currentTerm)
		rf.mu.Lock()
		if rf.isLeader {
			rf.mu.Unlock()
			time.Sleep(time.Duration(100) * time.Millisecond)
			//heartbeat send interval
			//time.Sleep(time.Duration(10) * time.Millisecond)
			//send heartbeats to all peers
			rf.mu.Lock()
			for peer := range rf.peers {
				if peer != rf.me && rf.isLeader {
					//// fmt.Printf("address of nextIndex %p\n", &rf.nextIndex[peer])
					// // fmt.Println("peer is ", peer, "me is ", rf.me)
					args := AppendEntriesArgs{}
					args.Entries = make([]LogEntry, 0)
					args.Entries = append(args.Entries, rf.log[rf.nextIndex[peer]:]...)
					args.Term = rf.currentTerm
					args.LeaderId = rf.me
					args.PrevLogIndex = rf.nextIndex[peer] - 1
					args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
					args.LeaderCommit = rf.commitIndex
					go func(i int, peerIndex int, logLength int) {
						reply := AppendEntriesReply{}
						ok := rf.sendAppendEntries(i, &args, &reply)
						if !ok {
							return
						}
						if reply.Success {
							rf.mu.Lock()
							// // fmt.Println(len(log))
							// fmt.Println("leader ", rf.me, "successfully sent appendentries with log ", rf.log)
							rf.nextIndex[i] = logLength
							rf.matchIndex[i] = logLength - 1
							rf.mu.Unlock()
						} else {
							//// fmt.Println("AppendEntries failed for ", i, " at term ", rf.currentTerm, " by ", rf.me)
							rf.mu.Lock()
							rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
							if reply.Term > rf.currentTerm {
								// fmt.Println("term mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
								rf.currentTerm = reply.Term
								rf.isInElection = false
								rf.isLeader = false
								rf.votedFor = -1
								rf.votesForMe = 0
								//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
								rf.persist()
								//add else if for incorrect term, just return do nothing
							} else if reply.Term != rf.currentTerm {
								rf.mu.Unlock()
								return
							} else {
								// handle prevLog mismatch
								// fmt.Println("prevLog mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
								if peerIndex > 1 {
									rf.nextIndex[i] = peerIndex - 1
								}
							}
							rf.mu.Unlock()
						}

					}(peer, rf.nextIndex[peer], len(rf.log))
				}
			}
			rf.mu.Unlock()
			commitChecker(rf)
		} else {
			rf.mu.Unlock()
			time.Sleep(time.Duration(10) * time.Millisecond)
			if rf.heartBeatExpired() && !rf.isLeader {
				rf.startElection()
				time.Sleep(time.Duration(500) * time.Millisecond)
				rf.mu.Lock()
				// fmt.Println("votes for me:", rf.votesForMe, "for", rf.me, " at term", rf.currentTerm)
				voteCheck := rf.votesForMe > len(rf.peers)/2
				rf.mu.Unlock()
				if voteCheck {
					//// fmt.Println("becoming leader by ", rf.me, " at term ", rf.currentTerm)
					rf.becomeLeader()
				}
			}
			sendOnApplyCh(rf)
		}
		//rf.mu.Unlock()

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

	}
}

// if this function is called, current raft node timer ran out and will be a candidate
func (rf *Raft) startElection() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm++
	// fmt.Println("startElection called by ", rf.me, " at term ", rf.currentTerm, "votes for me: ", rf.votesForMe, "log is ", rf.log)
	rf.votedFor = rf.me
	//this line is incorrect and causing issues with 4B, but changing it will break 4A and elections
	rf.votesForMe = 1
	rf.isInElection = true
	rf.isLeader = false
	rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
	//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
	rf.persist()
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		args := RequestVoteArgs{}
		args.Term = rf.currentTerm
		args.CandidateId = rf.me
		args.LastLogIndex = len(rf.log)
		args.LastLogTerm = rf.log[len(rf.log)-1].Term
		go func(i int, term int, log []LogEntry) {
			reply := RequestVoteReply{}
			//// fmt.Println("sending request vote to ", i, " at term ", rf.currentTerm)
			rf.sendRequestVote(i, &args, &reply)
			if reply.VoteGranted {
				rf.mu.Lock()
				rf.votesForMe++
				rf.mu.Unlock()
			} else {
				rf.mu.Lock()
				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.isInElection = false
					rf.isLeader = false
					rf.votedFor = -1
					rf.votesForMe = 0
					//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
					rf.persist()
				}
				rf.mu.Unlock()
			}
			//rf.mu.Unlock()
			//rf.currentTerm = reply.Term
		}(i, rf.currentTerm, rf.log)
	}
}

func (rf *Raft) heartBeatExpired() bool {
	rf.mu.Lock()
	currTime := int(time.Now().UnixMilli())-rf.timeOfLastHeartBeat > rf.heartBeatTimeout
	rf.mu.Unlock()
	return currTime
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rf *Raft) becomeLeader() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	fmt.Println("Node became leader: ", rf.me, " at term ", rf.currentTerm)
	rf.isLeader = true
	rf.isInElection = false
	rf.votesForMe = 0
	rf.heartBeatTimeout = rand.Intn(700) + 800
	//send a heartbeat to tell other nodes im the leader
	for peer := range rf.peers {
		if peer != rf.me && rf.isLeader {
			//// fmt.Printf("address of nextIndex %p\n", &rf.nextIndex[peer])
			// // fmt.Println("peer is ", peer, "me is ", rf.me)
			args := AppendEntriesArgs{}
			args.Entries = make([]LogEntry, 0)
			args.Entries = append(args.Entries, rf.log[rf.nextIndex[peer]:]...)
			args.Term = rf.currentTerm
			args.LeaderId = rf.me
			args.PrevLogIndex = rf.nextIndex[peer] - 1
			args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
			args.LeaderCommit = rf.commitIndex
			go func(i int, peerIndex int, logLength int) {
				reply := AppendEntriesReply{}
				ok := rf.sendAppendEntries(i, &args, &reply)
				if !ok {
					return
				}
				if reply.Success {
					rf.mu.Lock()
					// // fmt.Println(len(log))
					rf.nextIndex[i] = logLength
					rf.matchIndex[i] = logLength - 1
					rf.mu.Unlock()
				} else {
					//// fmt.Println("AppendEntries failed for ", i, " at term ", rf.currentTerm, " by ", rf.me)
					rf.mu.Lock()
					rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
					if reply.Term > rf.currentTerm {
						// fmt.Println("term mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
						rf.currentTerm = reply.Term
						rf.isInElection = false
						rf.isLeader = false
						rf.votedFor = -1
						rf.votesForMe = 0
						//fmt.Println("persistingdata with votedfor", rf.votedFor, "at term", rf.currentTerm, "for", rf.me, "with log", rf.log)
						rf.persist()
						//add else if for incorrect term, just return do nothing
					} else if reply.Term != rf.currentTerm {
						rf.mu.Unlock()
						return
					} else {
						// handle prevLog mismatch
						// fmt.Println("prevLog mismatch for ", i, " at term ", rf.currentTerm, " by ", rf.me)
						if peerIndex > 1 {
							rf.nextIndex[i] = peerIndex - 1
						}
					}
					rf.mu.Unlock()
				}

			}(peer, rf.nextIndex[peer], len(rf.log))
		}
	}

}

// if a majority of nodes have the same log entry, commit it by sending it out to the applyCh
func commitChecker(rf *Raft) {
	rf.mu.Lock()

	//// fmt.Println("commitChecker called by ", rf.me, " at term ", rf.currentTerm)
	//// fmt.Println("log, commitIndex, lastApplied", rf.log, rf.commitIndex, rf.lastApplied)

	for i := len(rf.log) - 1; i > rf.commitIndex; i-- {
		count := 1
		for j := range rf.peers {
			if rf.matchIndex[j] >= i {
				count++
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = i
			break
		}

	}
	rf.mu.Unlock()
	sendOnApplyCh(rf)
}

func sendOnApplyCh(rf *Raft) {
	rf.mu.Lock()
	if rf.commitIndex > rf.lastApplied {
		for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
			msg := ApplyMsg{true, rf.log[i].Command, i}
			//// fmt.Println("sending applymsg from ", rf.me, " at term ", rf.currentTerm, " with logs ", rf.log, " WITH MESSAGE: ", msg)
			rf.mu.Unlock()
			rf.applyCh <- msg
			rf.mu.Lock()
			rf.lastApplied += 1
			// fmt.Println("sending applymsg from ", rf.me, " at term ", rf.currentTerm, " with logs ", rf.log, " WITH MESSAGE: ", msg)
		}
	}
	rf.mu.Unlock()
}

// The service or tester wants to create a Raft server. The ports
// of all the Raft servers (including this one) are in peers[]. This
// server's port is peers[me]. All the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	fmt.Println("Make called by ", me)
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (4A, 4B).
	rf.mu = sync.Mutex{}
	rf.votedFor = -1
	rf.heartBeatTimeout = rand.Intn(700) + 800
	rf.timeOfLastHeartBeat = int(time.Now().UnixMilli())
	rf.isInElection = false
	rf.isLeader = false
	rf.votesForMe = 0
	rf.currentTerm = 0
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = 1
	}
	rf.matchIndex = make([]int, len(peers))
	for i := range rf.matchIndex {
		rf.matchIndex[i] = 0
	}
	rf.applyCh = applyCh

	// initialize from state persisted before a crash.
	if len(rf.log) < 1 {
		rf.log = make([]LogEntry, 1)
		rf.log[0] = LogEntry{0, 0}
	}
	// fmt.Println("reading persist for ", rf.me)
	rf.readPersist(persister.ReadRaftState())
	//fmt.Println("reading persist for ", rf.me, " with log ", rf.log)
	rf.persist()

	// start ticker goroutine to start elections.
	go rf.ticker()
	//// fmt.Println("Made a raft instance")

	return rf
}
