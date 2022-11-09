package main

import (
	"bufio"
	"context"
	"fmt"
	peer "github.com/AGmarsen/Handin-4/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

// states
const WANTED int = 0
const HELD int = 1
const RELEASED int = 2

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:      ownPort,
		clock:   0,
		state:   RELEASED,
		clients: make(map[int32]peer.VoteClient),
		ctx:     ctx,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	peerServer := peer.NewServer()
	peer.RegisterVoteServer(peerServer, p)

	go func() {
		if err := peerServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *peer.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := peer.Dial(fmt.Sprintf(":%v", port), peer.WithInsecure(), peer.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := peer.NewVoteClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		p.enter()
	}
}

type peer struct {
	peer.UnimplementedVoteServer
	id      int32
	clock   int32
	state   int
	clients map[int32]peer.VoteClient
	ctx     context.Context
	mutex   sync.Mutex
}

func (p *peer) Request(ctx context.Context, req *peer.Request) (*peer.Response, error) {
	updateLamport(req)
	log.Printf("Request received from %d (%d, %d)\n", req.id, p.id, p.clock)
	var resp peer.Response
	for p.state == HELD {
		//don't respond while in critical state
	}
	if p.state == RELEASED {
		resp = &peer.Response{id: p.id, p.clock}
		updateLamport(resp)
		resp = &peer.Response{id: p.id, p.clock}
		return resp, nil
	}
	//if wanted and request has higher priority
	if p.clock > req.clock || p.clock == req.clock && p.id > req.id {
		resp = &peer.Response{id: p.id, p.clock}
		updateLamport(resp)
		resp = &peer.Response{id: p.id, p.clock}
		return resp, nil
	}
	//else if I have higher priority
	for state != RELEASED {
		//don't respond until I'm are done
	}
	resp = &peer.Response{id: p.id, p.clock}
	updateLamport(resp)
	resp = &peer.Response{id: p.id, p.clock}
	return resp, nil
}

// try to enter critical state
func (p *peer) enter() {
	p.state = WANTED
	request := &peer.Request{id, clock}

	for id, client := range p.clients {
		updateLamport(request)
		log.Printf("Request sent to: %d (%d, %d)\n", id, p.id, p.clock)
		reply, err := client.Request(request)
		if err != nil {
			log.Printf(err)
		}
		updateLamport(reply)
		log.Printf("Reply recieved from %d (%d, %d)\n", reply.id, p.id, p.clock)
	}
	//after response from two peers
	p.state = HELD

	//Enter :-)
	doCriticalStuff()

	//Release Critical
	p.state = RELEASED

}

func doCriticalStuff() {
	log.Printf("I got permission :D")
}

func (p *Peer) updateLamport(req *peer.Request) {
	p.mutex.lock()
	defer p.mutex.unlock()
	req.clock = max(p.clock, req.clock) + 1
}

func (p *Peer) updateLamport(resp *peer.Response) {
	p.mutex.lock()
	defer p.mutex.unlock()
	req.clock = max(p.clock, resp.clock) + 1
}

func max(a int32, b int32) {
	if a > b {
		return a
	}
	return b
}
