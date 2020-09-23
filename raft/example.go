package raft

import (
	"log"
	"sync"
	"testing"
)

type Engine struct {
	mu sync.Mutex

	cluster []*Server     // raft节点集群
	storage []*MapStorage // 存储服务

	commitChans []chan CommitEntry // server committed log to chan
	commits     [][]CommitEntry    //

	connected []bool //
	alive     []bool

	n int // raft节点个数
	t *testing.T
}

func NewEngine(t *testing.T, n int) *Engine {
	serverNum := make([]*Server, n)
	connected := make([]bool, n)
	alive := make([]bool, n)
	commitChans := make([]chan CommitEntry, n)
	commits := make([][]CommitEntry, n)
	ready := make(chan interface{})
	storage := make([]*MapStorage, n)

	// 依次启动每一个服务
	for i := 0; i < n; i++ {
		peerIds := make([]int, 0)
		for p := 0; p < n; p++ {
			if p != i {
				peerIds = append(peerIds, p)
			}
		}
		storage[i] = NewMapStorage()
		commitChans[i] = make(chan CommitEntry)
		serverNum[i] = NewServer(i, peerIds, storage[i], commitChans[i], ready)
		serverNum[i].Server()
		alive[i] = true
	}

	// 相互连接
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j {
				serverNum[i].ConnectToPeer(j, serverNum[i].GetListenAddr())
			}
		}
		connected[i] = true
	}

	close(ready)

	e := &Engine{
		cluster:     serverNum,
		storage:     storage,
		commitChans: commitChans,
		commits:     commits,
		connected:   connected,
		alive:       alive,
		n:           n,
		t:           t,
	}

	for i := 0; i < n; i++ {
		go e.CollectCommits(i)
	}

	return e
}

// ShutDown 关闭集群中的所有server
func (e *Engine) ShutDown() {
	// 1. 取消所有连接
	for i := 0; i < e.n; i++ {
		e.cluster[i].DisconnectAll()
		e.connected[i] = false
	}
	// 2. 关闭所有 server
	for i := 0; i < e.n; i++ {
		if e.alive[i] {
			e.alive[i] = false
			e.cluster[i].ShutDown()
		}
	}
	// 3. 关闭所有 commit chan
	for i := 0; i < e.n; i++ {
		close(e.commitChans[i])
	}
}

// DisconnectPeer 断开某个节点与其他所有节点的连接
func (e Engine) DisconnectPeer(id int) {
	tlog("Disconnect %d", id)
	e.cluster[id].DisconnectAll()
	for i := 0; i < e.n; i++ {
		if i != id {
			e.cluster[i].DisconnectPeer(id)
		}
	}
	e.connected[i] = false
}

// CollectCommits 从commit chan 收集 commit 信息
func (e *Engine) CollectCommits(i int) {
	for c := range e.commitChans[i] {
		e.mu.Lock()
		tlog("collectCommits(%d) got %+v", i, c)
		e.commits[i] = append(e.commits[i], c)
		e.mu.Unlock()
	}
}

func tlog(format string, a ...interface{}) {
	format = "[TEST] " + format
	log.Printf(format, a...)
}
