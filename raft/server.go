package raft

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// Server 为 raft 节点提供 RPC 服务
type Server struct {
	mu sync.Mutex

	serverID int   // server id
	peerIds  []int // peers id

	cm       *ConsensusModule // 共识模块
	storage  Storage          // 存储
	rpcProxy *RPCProxy

	rpcServer *rpc.Server  // rpc server
	listener  net.Listener // 监听端口

	commitChan  chan<- CommitEntry  // log commited chan
	peerClients map[int]*rpc.Client // peers rpc client

	ready <-chan interface{}
	quit  chan interface{}
	wg    *sync.WaitGroup
}

// NewServer a new rpc server
func NewServer(
	serverID int,
	peerIds []int,
	storage Storage,
	commitChan chan<- CommitEntry,
	ready <-chan interface{}) *Server {
	return &Server{
		serverID:    serverID,
		peerIds:     peerIds,
		storage:     storage,
		commitChan:  commitChan,
		peerClients: make(map[int]*rpc.Client),
		ready:       ready,
		quit:        make(chan interface{}),
	}
}

// Server run a rpc server
// 1. register
// 2. listen port
// 3. client.Call
func (s *Server) Server() {
	s.mu.Lock()
	s.cm = NewConsensusModule(s.serverID, s.peerIds, s, s.storage, s.ready, s.commitChan)

	// create rpc server
	s.rpcServer = rpc.NewServer()
	s.rpcProxy = &RPCProxy{cm: s.cm}
	// register
	s.rpcServer.RegisterName("ConsensusModule", s.rpcProxy)

	var err error
	// listen the port
	// use ':0' 代表自动选择 port，可使用 listener.Addr() 获取地址
	s.listener, err = net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[%v] listening at %s", s.serverID, s.listener.Addr())
	s.mu.Unlock()

	// 开始 rpc 服务
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// 监听rpc client请求
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				// 如果是 quit 信号，退出
				case <-s.quit:
					return
				// 否则报错
				default:
					log.Fatal("accept error:", err)
				}
			}

			s.wg.Add(1)
			go func() {
				s.rpcServer.ServeConn(conn)
				s.wg.Done()
			}()
		}

	}()

}

// Call 代表一次rpc服务调用过程，rpc client call
func (s *Server) Call(id int, serviceName string, args interface{}, reply interface{}) error {
	s.mu.Lock()
	peer := s.peerClients[id]
	s.mu.Unlock()

	if peer == nil {
		return fmt.Errorf("call client %d after it's closed", id)
	} else {
		return peer.Call(serviceName, args, reply)
	}

}

// ShutDown 关闭rpc server
func (s *Server) ShutDown() {
	s.cm.Stop()        // 共识模块关闭
	close(s.quit)      // 关闭quit chan
	s.listener.Close() // 关闭端口监听
	s.wg.Wait()        // 等待 wg 为 0 然后关闭
}

// DisconnectAll 断开该节点与所有客户端的连接
func (s *Server) DisconnectAll() {
	s.mu.Lock()

	for id := range s.peerClients {
		if s.peerClients[id] != nil {
			s.peerClients[id].Close()
			s.peerClients[id] = nil
		}
	}

	s.mu.Unlock()
}

// GetListenAddr 获取server 监听的端口
func (s *Server) GetListenAddr() net.Addr {
	s.mu.Lock()
	addr := s.listener.Addr()
	s.mu.Unlock()
	return addr
}

// ConnectToPeer 连接到 peer
func (s *Server) ConnectToPeer(peerId int, addr net.Addr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] == nil {
		client, err := rpc.Dial(addr.Network(), addr.String())
		if err != nil {
			return err
		}
		s.peerClients[peerId] = client
	}
	return nil
}

//DisconnectPeer 断开连接
func (s *Server) DisconnectPeer(peerId int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] != nil {
		err := s.peerClients[peerId].Close()
		s.peerClients[peerId] = nil
		return err
	}
	return nil
}

// RPCProxy 是对 consensus module 相关rpc方法(RV和AE)的一个代理
// 这里使用proxy是为了模拟一些不可靠的连接，随机丢弃rpc连接或者延迟连接时间
// 更接近真实环境
// 正常情况下应该这样注册 cm: s.rpcServer.RegisterName("ConsensusModule", s.cm)
type RPCProxy struct {
	cm *ConsensusModule
}

// RequestVote 请求投票 rpc proxy
func (rpp *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			rpp.cm.dlog("drop RequestVote")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			rpp.cm.dlog("delay RequestVote")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.cm.RequestVote(args, reply)
}

// AppendEntries append log entries rpc proxy
func (rpp *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			rpp.cm.dlog("drop AppendEntries")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			rpp.cm.dlog("delay AppendEntries")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.cm.AppendEntries(args, reply)
}
