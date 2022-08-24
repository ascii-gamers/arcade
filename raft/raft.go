package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"arcade/arcade/message"
	"arcade/arcade/net"
	"arcade/labgob"
)

type RaftState int

const (
	batchInterval      = 10 * time.Millisecond
	heartbeatInterval  = 100
	electionTimeoutMin = 600
	electionTimeoutMax = 900

	NullPeer = -1

	Follower RaftState = iota
	Candidate
	Leader
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid    bool
	Command         interface{}
	CommandIndex    int
	CommandTimestep int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	sync.RWMutex               // Lock to protect shared access to this peer's state
	peers        []*net.Client // RPC end points of all peers
	network      *net.Network
	persister    *Persister // Object to hold this peer's persisted state
	me           int        // this peer's index into peers[]
	dead         int32      // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	log             *Log
	start           time.Time
	state           RaftState
	batchTimer      *time.Timer
	electionTimeout time.Time
	commitIndex     int
	lastApplied     int

	applyCh   chan ApplyMsg
	commitMux sync.Mutex

	currentTerm int
	votedFor    int

	nextIndex  []int
	matchIndex []int

	currentLeader int

	timestepPeriod int
	timestep       int
	timestepCond   *sync.Cond
}

func (rf *Raft) print(function, message string) {
	// fmt.Printf("RAFT %dms [%s id=%d term=%d] %s\n", time.Since(rf.start).Milliseconds(), function, rf.me, rf.currentTerm, message)
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.RLock()
	defer rf.RUnlock()

	return rf.currentTerm, rf.state == Leader
}

func (rf *Raft) GetLog() (Log, int, int) {
	rf.RLock()
	defer rf.RUnlock()
	return *rf.log, rf.lastApplied, rf.commitIndex
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
// Lock must already be held
func (rf *Raft) persist(snapshot []byte) {
	w := new(bytes.Buffer)

	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log.GetLastIncludedIndex())
	e.Encode(rf.log.GetLastIncludedTerm())
	e.Encode(rf.log.GetEntries())

	// if snapshot == nil {
	// 	rf.persister.SaveRaftState(w.Bytes())
	// } else {
	// 	rf.persister.SaveStateAndSnapshot(w.Bytes(), snapshot)
	// }
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 {
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)

	rf.Lock()
	defer rf.Unlock()

	var currentTerm int
	d.Decode(&currentTerm)
	rf.currentTerm = currentTerm

	var votedFor int
	d.Decode(&votedFor)
	rf.votedFor = votedFor

	var lastIncludedIndex int
	d.Decode(&lastIncludedIndex)
	rf.log.SetLastIncludedIndex(lastIncludedIndex)

	var lastIncludedTerm int
	d.Decode(&lastIncludedTerm)
	rf.log.SetLastIncludedTerm(lastIncludedTerm)

	var entries []LogEntry
	d.Decode(&entries)
	rf.log.SetEntries(entries)
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	rf.Lock()
	defer rf.Unlock()

	rf.print("Snapshot", "")

	if index <= rf.log.GetLastIncludedIndex() {
		return
	}

	rf.print("Snapshot", "Applying snapshot at index "+strconv.Itoa(index))

	lastEntry, ok := rf.log.GetEntry(index)

	if !ok {
		panic("Cannot find entry at index " + strconv.Itoa(index))
	}

	err := rf.log.DeleteEntriesPreceding(index + 1)

	if err != nil || index != lastEntry.Index {
		panic(err)
	}

	rf.print("Snapshot", "lastIncludedIndex="+strconv.Itoa(lastEntry.Index)+" lastIncludedTerm="+strconv.Itoa(lastEntry.Term))

	rf.log.SetLastIncludedIndex(lastEntry.Index)
	rf.log.SetLastIncludedTerm(lastEntry.Term)
	rf.persist(snapshot)
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	message.Message
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
	ClientId     int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	message.Message
	Term        int
	VoteGranted bool
	ClientId    int
}

func (m RequestVoteArgs) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m RequestVoteReply) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs) *RequestVoteReply {

	rf.Lock()
	defer rf.Unlock()
	reply := RequestVoteReply{message.Message{Type: "RequestVoteReply"}, rf.currentTerm, false, rf.me}

	// defer func() {

	// }()

	if args.Term < rf.currentTerm {
		log.Println("[RAFT]: RequestVote", "Reject, args.Term < currentTerm")
		return &reply
	} else if args.Term > rf.currentTerm {
		rf.startNewTerm(args.Term, NullPeer, NullPeer)
		rf.persist(nil)
	}

	if rf.votedFor != NullPeer && rf.votedFor != args.CandidateID {
		log.Println("[RAFT]: RequestVote", "Reject, already voted for "+strconv.Itoa(rf.votedFor))
		return &reply
	}

	if rf.log.LastTerm() > args.LastLogTerm || (rf.log.LastTerm() == args.LastLogTerm && rf.log.LastIndex() > args.LastLogIndex) {
		log.Println("[RAFT]: RequestVote", "Reject, log not up to date")
		return &reply
	}

	rf.resetElectionTimeout()

	rf.votedFor = args.CandidateID
	rf.persist(nil)

	log.Println("[RAFT]: RequestVote", "Voted for "+strconv.Itoa(args.CandidateID))

	reply.VoteGranted = true

	return &reply
}

//
// AppendEntries RPC arguments structure
//
type AppendEntriesArgs struct {
	message.Message
	// Leader's term
	Term int

	// So follower can redirect clients
	ClientId int

	// Index of log entry immediately preceding new ones
	PrevLogIndex int

	// Term of prevLogIndex entry
	PrevLogTerm int

	// Log entries to store (empty for heartbeat; may send more than one for efficiency)
	Entries []LogEntry

	// Leader's commitIndex
	LeaderCommit int

	Timestep int
}

//
// AppendEntries RPC reply structure
//
type AppendEntriesReply struct {
	message.Message
	Term          int
	Success       bool
	ConflictIndex int
	ClientId      int
}

func (m AppendEntriesArgs) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m AppendEntriesReply) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

//
// AppendEntries RPC handler
//
func (rf *Raft) AppendEntries(args *AppendEntriesArgs) *AppendEntriesReply {
	rf.Lock()
	defer rf.Unlock()

	reply := &AppendEntriesReply{Message: message.Message{Type: "AppendEntriesReply"}}
	reply.ClientId = rf.me

	defer func() {
		reply.Term = rf.currentTerm
	}()

	reply.ConflictIndex = rf.log.LastIndex() + 1

	// 1
	if args.Term < rf.currentTerm {
		log.Println("[RAFT]", "AppendEntries", "Term too smol")
		return reply
	}

	if args.Term > rf.currentTerm || rf.state == Candidate {
		log.Println("[RAFT]", "AppendEntries", "New term: "+strconv.Itoa(rf.currentTerm))
		rf.startNewTerm(args.Term, NullPeer, args.ClientId)
		rf.persist(nil)
	}

	// log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Received %d entries, prevLogIndex=%d, conflictIndex=%d", len(args.Entries), args.PrevLogIndex, reply.ConflictIndex))

	rf.resetElectionTimeout()

	// 2
	entry, ok := rf.log.GetEntry(args.PrevLogIndex)
	needsConflictIndexUpdate := false

	if ok {
		// log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Entry index: %d, entry term: %d", entry.Index, entry.Term))
		needsConflictIndexUpdate = entry.Term != args.PrevLogTerm
	} else if args.PrevLogIndex <= rf.log.GetLastIncludedIndex() && args.PrevLogTerm == rf.log.GetLastIncludedTerm() {
		// log.Println("[RAFT]", "AppendEntries", "Contains last included entry")
	} else if args.PrevLogIndex != 0 {
		log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Log doesn't contain entry at index %d with term %d, lastIndex=%d, lastTerm=%d", args.PrevLogIndex, args.PrevLogTerm, rf.log.LastIndex(), rf.log.LastTerm()))
		log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("%v", rf.log.GetEntries()))
		return reply
	} else if args.PrevLogIndex < 0 {
		panic("prevLogIndex should never be negative")
	}

	if needsConflictIndexUpdate {
		log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Failed condition 2, cmd=%v, idx=%d, %d != %d", entry.Command, entry.Index, entry.Term, args.PrevLogTerm))

		rf.log.Iter(func(e LogEntry) bool {
			if e.Term == entry.Term {
				reply.ConflictIndex = e.Index
				return false
			}

			return true
		})

		log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Conflict index: %d", reply.ConflictIndex))

		return reply
	}

	rf.timestep = args.Timestep
	reply.Success = true

	// 3
	needsPersist := false

	for _, entry := range args.Entries {
		if existingEntry, ok := rf.log.GetEntry(entry.Index); ok && entry.Term != existingEntry.Term {
			rf.log.DeleteEntryAndFollowing(entry.Index)
			needsPersist = true
		}
	}

	// 4
	for _, entry := range args.Entries {
		if _, ok := rf.log.GetEntry(entry.Index); !ok && entry.Index > rf.log.LastIndex() {
			rf.log.AppendEntry(entry)
			needsPersist = true
			// log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Appending, %v, %d", entry.Command, entry.Index))
		}
	}

	if needsPersist {
		rf.persist(nil)
	}

	// 5
	if args.LeaderCommit > rf.commitIndex {
		oldCommitIndex := rf.commitIndex
		rf.commitIndex = min(args.LeaderCommit, rf.log.LastIndex())
		log.Println("[RAFT]", "AppendEntries", fmt.Sprintf("Updated commit index from %d to %d", oldCommitIndex, rf.commitIndex))
		go rf.commit()
	}

	rf.startNewTerm(args.Term, args.ClientId, args.ClientId)
	rf.persist(nil)

	return reply
}

//
// InstallSnapshot RPC arguments structure
//
type InstallSnapshotArgs struct {
	message.Message
	// Leader's term
	Term int

	// So follower can redirect clients
	ClientId int

	// The snapshot replaces all entries up through and including this index
	LastIncludedIndex int

	// Term of lastIncludedIndex
	LastIncludedTerm int

	// Raw bytes of the snapshot chunk, starting at offset
	Data []byte
}

//
// InstallSnapshot RPC reply structure
//
type InstallSnapshotReply struct {
	message.Message
	Term     int
	ClientId int
}

func (m InstallSnapshotArgs) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m InstallSnapshotReply) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

//
// InstallSnapshot RPC handler
//

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs) *InstallSnapshotReply {

	rf.Lock()
	defer rf.Unlock()

	reply := &InstallSnapshotReply{Message: message.Message{Type: "RequestVote"}}

	reply.ClientId = rf.me

	defer func() {
		reply.Term = rf.currentTerm
	}()

	if args.Term < rf.currentTerm {
		rf.print("InstallSnapshot", "Term too smol")
		return reply
	}

	if args.Term > rf.currentTerm || rf.state == Candidate {
		rf.print("InstallSnapshot", "New term: "+strconv.Itoa(rf.currentTerm))
		rf.startNewTerm(args.Term, NullPeer, args.ClientId)
	}

	rf.print("InstallSnapshot", fmt.Sprintf("%v", rf.log.entries))
	rf.print("InstallSnapshot", fmt.Sprintf("Received snapshot, %d bytes, lastIncludedIndex=%d, lastIncludedTerm=%d", len(args.Data), args.LastIncludedIndex, args.LastIncludedTerm))
	rf.resetElectionTimeout()

	// Ignore 2-5

	// 6
	// entry, ok := rf.log.GetEntry(args.LastIncludedIndex)

	// if (ok && entry.Term == args.LastIncludedTerm) || (args.LastIncludedIndex == rf.log.GetLastIncludedIndex() && args.LastIncludedTerm == rf.log.GetLastIncludedTerm()) {
	if args.LastIncludedIndex < rf.log.LastIndex() {
		rf.log.DeleteEntriesPreceding(args.LastIncludedIndex + 1)

		if len(rf.log.entries) > 0 {
			rf.log.SetLastIncludedIndex(rf.log.entries[0].Index - 1)
			rf.log.SetLastIncludedTerm(rf.log.entries[0].Term)
		} else {
			rf.log.SetLastIncludedIndex(args.LastIncludedIndex)
			rf.log.SetLastIncludedTerm(args.LastIncludedTerm)
		}

		rf.persist(args.Data)
		rf.print("InstallSnapshot", fmt.Sprintf("Deleted entries preceding %d -- %v", args.LastIncludedIndex+1, rf.log.entries))
		return reply
	}

	// rf.print("InstallSnapshot", fmt.Sprintf("%t %d %d", ok, entry.Term, args.LastIncludedTerm))

	// 7
	rf.log = &Log{}
	rf.log.SetLastIncludedIndex(args.LastIncludedIndex)
	rf.log.SetLastIncludedTerm(args.LastIncludedTerm)
	rf.persist(args.Data)

	rf.print("InstallSnapshot", fmt.Sprintf("Cleared log -- %v", rf.log.entries))

	go rf.commit()

	return reply
}

type ForwardedStartArgs struct {
	message.Message
	ClientId int
	Command  interface{}
}

type ForwardedStartReply struct {
	message.Message
	ClientId int
	Index    int
	Term     int
}

func (m ForwardedStartArgs) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m ForwardedStartReply) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.Lock()
	defer rf.Unlock()

	if rf.state != Leader {
		// log.Println("[RAFT]", "currentLeader", rf.currentLeader)
		if rf.currentLeader >= 0 {
			args := &ForwardedStartArgs{Message: message.Message{Type: "ForwardedStart"}, ClientId: rf.me, Command: command}
			leader := rf.peers[rf.currentLeader]
			// log.Println("[RAFT]", "currentLeader2", leader.ID)
			log.Println("[RAFT]", "sending forwardedstart")
			if reply, err := rf.network.SendAndReceive(leader, args); err == nil {
				reply := reply.(ForwardedStartReply)
				return reply.Index, reply.Term, false
			}
		}
		return -1, -1, false
	}

	rf.print("Start", fmt.Sprintf("Starting with command %v", command))

	entry := LogEntry{rf.currentTerm, rf.log.LastIndex() + 1, command, rf.timestep}
	rf.print("Start", fmt.Sprintf("Appending entry with index %d", entry.Index))
	rf.log.AppendEntry(entry)
	rf.persist(nil)

	go rf.sendAllAppendEntriesBatched()

	return entry.Index, rf.currentTerm, true
}

func (rf *Raft) ForwardedStart(args *ForwardedStartArgs) *ForwardedStartReply {
	log.Println("[RAFT]", "rec forwardedstart")
	if rf.state == Leader {
		ind, term, _ := rf.Start(args.Command)
		return &ForwardedStartReply{message.Message{Type: "ForwardedStartReply"}, rf.me, ind, term}
	}
	return &ForwardedStartReply{message.Message{Type: "ForwardedStartReply"}, rf.me, -1, -1}
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// Lock must already be held
func (rf *Raft) startNewTerm(term int, peer int, leader int) {
	rf.currentTerm = term
	rf.state = Follower
	rf.votedFor = peer
	rf.currentLeader = leader
}

// Lock must already be held
func (rf *Raft) resetElectionTimeout() {
	// Set electionTimeout at random duration within range from now
	randomDuration := electionTimeoutMin + rand.Float64()*(electionTimeoutMax-electionTimeoutMin)
	//log.Println("[RAFT]:", "random duration", randomDuration)
	rf.electionTimeout = time.Now().Add(time.Duration(randomDuration) * time.Millisecond)

	// Call electionTicker at electionTimeout plus one millisecond
	time.AfterFunc(time.Until(rf.electionTimeout.Add(time.Millisecond)), rf.electionTicker)
}

func (rf *Raft) StartTime() {
	rf.startTimestepCounter()
}

func (rf *Raft) startTimestepCounter() {
	go func() {
		for {
			rf.Lock()
			rf.timestep += 1
			rf.Unlock()
			rf.timestepCond.Broadcast()
			time.Sleep(time.Duration(rf.timestepPeriod) * time.Millisecond)
		}
	}()
}

func (rf *Raft) GetTimestep() int {
	rf.RLock()
	defer rf.RUnlock()
	return rf.timestep
}

func (rf *Raft) runElection() {
	rf.Lock()
	defer rf.Unlock()

	rf.startNewTerm(rf.currentTerm+1, rf.me, rf.me)
	rf.state = Candidate
	rf.persist(nil)

	log.Println("[RAFT]:", "runElection", "New term: "+strconv.Itoa(rf.currentTerm))

	args := &RequestVoteArgs{
		Message:      message.Message{Type: "RequestVote"},
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: rf.log.LastIndex(),
		LastLogTerm:  rf.log.LastTerm(),
		ClientId:     rf.me,
	}

	votes := make(chan *RequestVoteReply, len(rf.peers))

	rf.print("runElection", "Requesting votes")

	for i, peer := range rf.peers {
		if i == rf.me {
			votes <- &RequestVoteReply{message.Message{Type: "RequestVoteReply"}, rf.currentTerm, true, rf.me}
			continue
		}

		go func(votes chan *RequestVoteReply, peer *net.Client) {
			if reply, err := rf.network.SendAndReceive(peer, args); err == nil {
				log.Println("[RAFT]:", "RECEIVED vote", reply)
				reply := reply.(RequestVoteReply)
				votes <- &reply
			} else {
				log.Println("[RAFT]:", "vote error", err)
			}
		}(votes, peer)
	}

	voteCount := 0

	for range rf.peers {
		go func() {
			reply := <-votes

			if rf.killed() {
				return
			}

			rf.Lock()
			defer rf.Unlock()

			log.Println("[RAFT]:", "runElection", fmt.Sprintf("Received vote: %t, %d", reply.VoteGranted, reply.ClientId))

			if rf.state != Candidate || rf.currentTerm != args.Term {
				log.Println("[RAFT]:", "runElection, return", "bruh wut")
				return
			}

			if reply.Term > rf.currentTerm {
				log.Println("[RAFT]:", "runElection, return", "overriden")
				rf.startNewTerm(reply.Term, NullPeer, reply.ClientId)
				rf.persist(nil)
				return
			}

			if !reply.VoteGranted {
				return
			}

			voteCount++

			if voteCount <= len(rf.peers)/2 {
				log.Println("[RAFT]:", "runElection, return", "not enough")
				return
			}

			if rf.state == Leader {
				// Already won election
				return
			}

			log.Println("[RAFT]:", "runElection", "Won election!")

			rf.state = Leader
			rf.nextIndex = make([]int, len(rf.peers))
			rf.matchIndex = make([]int, len(rf.peers))

			for i := range rf.peers {
				rf.nextIndex[i] = rf.log.LastIndex() + 1
			}

			go rf.heartbeatTicker()
		}()
	}
}

func (rf *Raft) electionTicker() {
	if rf.killed() {
		return
	}

	rf.Lock()
	defer rf.Unlock()

	if time.Now().Before(rf.electionTimeout) {
		return
	}

	if rf.state == Follower || rf.state == Candidate {
		go rf.runElection()
	}

	rf.resetElectionTimeout()
}

func (rf *Raft) sendAllAppendEntriesBatched() {
	rf.Lock()
	defer rf.Unlock()

	if rf.killed() {
		return
	}

	if rf.batchTimer != nil {
		rf.batchTimer.Stop()
	}

	rf.print("sendAllAppendEntriesBatched", "Starting batch timeout")

	rf.batchTimer = time.AfterFunc(batchInterval, func() {
		rf.RLock()
		rf.print("sendAllAppendEntriesBatched", "Sending batch")
		rf.RUnlock()

		rf.sendAllAppendEntries()
	})
}

func (rf *Raft) sendAllAppendEntries() {
	rf.Lock()
	defer rf.Unlock()

	if rf.killed() || rf.state != Leader {
		return
	}

	rf.matchIndex[rf.me] = rf.log.LastIndex()

	for server, peer := range rf.peers {
		if server == rf.me {
			continue
		}

		rf.sendAppendEntries(server, peer)
	}
}

// Must hold lock
func (rf *Raft) sendAppendEntries(server int, peer *net.Client) {
	entries := rf.log.GetEntryAndFollowing(rf.nextIndex[server])
	prevLogIndex := rf.nextIndex[server] - 1
	prevLogTerm := rf.currentTerm

	if prevLogIndex < rf.log.GetLastIncludedIndex() {
		args := &InstallSnapshotArgs{
			Message:           message.Message{Type: "InstallSnapshot"},
			Term:              rf.currentTerm,
			ClientId:          rf.me,
			LastIncludedIndex: rf.log.GetLastIncludedIndex(),
			LastIncludedTerm:  rf.log.GetLastIncludedTerm(),
			// Data:              rf.persister.ReadSnapshot(),
		}

		rf.print("sendAppendEntries", fmt.Sprintf("Sending InstallSnapshot to %d, prevLogIndex=%d, lastIncludedIndex=%d, lastIncludedTerm=%d, size=%d", server, prevLogIndex, rf.log.GetLastIncludedIndex(), rf.log.GetLastIncludedTerm(), len(args.Data)))

		go func() {
			if _, err := rf.network.SendAndReceive(peer, args); err == nil {
				return
			}

			rf.Lock()
			defer rf.Unlock()

			if rf.killed() || rf.state != Leader || rf.currentTerm != args.Term {
				return
			}

			rf.nextIndex[server] = args.LastIncludedIndex + 1
			rf.matchIndex[server] = args.LastIncludedIndex
		}()

		return
	}

	// Don't send AppendEntries if no entries to send
	if len(entries) > 0 {
		prevLogIndex = entries[0].Index - 1
		rf.print("sendAppendEntries", fmt.Sprintf("Sending %d entries to %d, nextIndex=%d, log last index=%d", len(entries), server, rf.nextIndex[server], rf.log.LastIndex()))
	}

	if prevEntry, ok := rf.log.GetEntry(prevLogIndex); ok {
		prevLogTerm = prevEntry.Term
	} else if prevLogIndex == rf.log.GetLastIncludedIndex() {
		prevLogTerm = rf.log.GetLastIncludedTerm()
	}

	args := &AppendEntriesArgs{
		Message:      message.Message{Type: "AppendEntries"},
		Term:         rf.currentTerm,
		ClientId:     rf.me,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
		Timestep:     rf.timestep,
	}

	go func() {
		// log.Println("[RAFT]", "AppendEntries", len(args.Entries))
		reply, err := rf.network.SendAndReceive(peer, args)

		if err != nil || len(entries) == 0 {
			return
		}

		if reply, ok := reply.(AppendEntriesReply); ok {
			rf.Lock()
			defer rf.Unlock()

			// log.Println("[RAFT]", "AppendEntries reply", reply)

			if rf.killed() || rf.state != Leader || rf.currentTerm != args.Term {
				return
			}

			if reply.Term > rf.currentTerm {
				rf.startNewTerm(reply.Term, NullPeer, reply.ClientId)
				rf.persist(nil)
				return
			}

			if reply.Success {
				// log.Println("[RAFT]", "AppendEntries", "advance index")
				rf.matchIndex[server] = args.PrevLogIndex + len(entries)
				rf.nextIndex[server] = rf.matchIndex[server] + 1
				rf.print("sendAppendEntries", fmt.Sprintf("Success, updating nextIndex[%d] to %d", server, rf.nextIndex[server]))
			} else {
				// log.Println("[RAFT]", "AppendEntries", "conflict")
				rf.nextIndex[server] = reply.ConflictIndex
			}

			rf.updateCommitIndex()
		}
	}()
}

func (rf *Raft) heartbeatTicker() {
	if rf.killed() {
		return
	}

	rf.Lock()
	defer rf.Unlock()

	if rf.state != Leader {
		return
	}

	go rf.sendAllAppendEntriesBatched()

	time.AfterFunc(heartbeatInterval*time.Millisecond, rf.heartbeatTicker)
}

// Lock must already be held
func (rf *Raft) updateCommitIndex() {
	if rf.state != Leader {
		return
	}

	rf.print("updateCommitIndex", fmt.Sprintf("matchIndex=%d commitIndex=%d, lastIndex=%d", rf.matchIndex, rf.commitIndex, rf.log.LastIndex()))

	for i := rf.log.LastIndex(); i > rf.commitIndex; i-- {
		count := 0

		for j := range rf.matchIndex {
			if rf.matchIndex[j] >= i {
				count++
			}
		}

		if count <= len(rf.peers)/2 {
			continue
		}

		entry, ok := rf.log.GetEntry(i)

		if !ok || entry.Term != rf.currentTerm {
			rf.print("updateCommitIndex", fmt.Sprintf("entry.Term = %d, rf.currentTerm = %d", entry.Term, rf.currentTerm))
			continue
		}

		log.Println("updateCommitIndex", fmt.Sprintf("Updating commit index to %d", i))
		rf.commitIndex = i
		go rf.commit()

		break
	}
}

func (rf *Raft) commit() {
	rf.commitMux.Lock()
	defer rf.commitMux.Unlock()

	for {
		rf.Lock()
		index := rf.lastApplied + 1
		rf.print("commit", fmt.Sprintf("index=%d, commitIndex=%d, lastIncludedIndex=%d", index, rf.commitIndex, rf.log.GetLastIncludedIndex()))

		if index > rf.commitIndex {
			rf.Unlock()
			break
		}

		if index <= rf.log.GetLastIncludedIndex() {
			rf.applyCh <- ApplyMsg{
				SnapshotValid: true,
				// Snapshot:      rf.persister.ReadSnapshot(),
				SnapshotTerm:  rf.log.GetLastIncludedTerm(),
				SnapshotIndex: rf.log.GetLastIncludedIndex(),
			}

			rf.print("commit", fmt.Sprintf("%v", rf.log.entries))
			rf.print("commit", fmt.Sprintf("Applying snapshot, %d", rf.log.GetLastIncludedIndex()))

			rf.lastApplied = rf.log.GetLastIncludedIndex()
			rf.Unlock()

			continue
		}

		entry, ok := rf.log.GetEntry(index)

		if !ok {
			fmt.Printf("%v\n", entry)
			fmt.Printf("%v\n", rf.log.GetEntries())
			panic(fmt.Sprintf("[id=%d] Entry doesn't exist: index=%d, lastApplied=%d, lastIndex=%d, lastIncludedIndex=%d", rf.me, index, rf.lastApplied, rf.log.LastIndex(), rf.log.GetLastIncludedIndex()))
		}

		rf.lastApplied = index
		rf.print("commit", fmt.Sprintf("Applying, %d", index))
		rf.Unlock()

		rf.applyCh <- ApplyMsg{
			CommandValid:    true,
			Command:         entry.Command,
			CommandIndex:    index,
			CommandTimestep: entry.Timestep,
		}
	}
}

func (rf *Raft) GetPersistentSize() int {
	return rf.persister.RaftStateSize()
}

func (rf *Raft) GetLogLastIndex() int {
	rf.RLock()
	defer rf.RUnlock()

	return rf.log.LastIndex()
}

func (rf *Raft) processMessage(from interface{}, data interface{}) interface{} {
	// log.Println("IN PROCESSMESSAGE: ", data)
	// c := from.(*net.Client)
	switch data := data.(type) {
	case RequestVoteArgs:
		return rf.RequestVote(&data)
	case AppendEntriesArgs:
		defer func() {
			// log.Println("[RAFT]", "resulting log len", len(rf.log.entries))
		}()
		return rf.AppendEntries(&data)
	case InstallSnapshotArgs:
		return rf.InstallSnapshot(&data)
	case ForwardedStartArgs:
		return rf.ForwardedStart(&data)

	}

	return nil
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*net.Client, me int, applyCh chan ApplyMsg, network *net.Network, timestepPeriod int, timestepCond *sync.Cond) *Raft {
	rand.Seed(time.Now().UnixNano())

	rf := &Raft{}
	rf.peers = peers
	// rf.persister = persister
	rf.me = me
	rf.network = network

	// Initialization
	rf.log = &Log{}
	rf.start = time.Now()
	rf.votedFor = NullPeer
	rf.state = Follower
	rf.applyCh = applyCh
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	rf.currentLeader = -1

	rf.timestepPeriod = timestepPeriod
	rf.timestep = 0
	rf.timestepCond = timestepCond

	// start ticker goroutine to start elections
	rf.resetElectionTimeout()

	// initialize from state persisted before a crash
	// rf.readPersist(persister.ReadRaftState())
	rf.commitIndex = rf.log.GetLastIncludedIndex()
	rf.lastApplied = rf.log.GetLastIncludedIndex()

	message.AddListener(rf.processMessage)

	return rf
}
