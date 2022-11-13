package main

import (
	"bufio"
	"context"
	"fmt"
	voter "github.com/AGmarsen/Handin-4/proto"
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
		clients: make(map[int32]voter.VoteClient),
		ctx:     ctx,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	peerServer := grpc.NewServer()
	voter.RegisterVoteServer(peerServer, p)

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

		var conn *grpc.ClientConn
		log.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := voter.NewVoteClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() { //press enter to request critical state
		p.enter()
	}
}

type peer struct {
	voter.UnimplementedVoteServer
	id      int32
	clock   int32
	state   int
	clients map[int32]voter.VoteClient
	ctx     context.Context
	mutex   sync.Mutex
}

func (p *peer) Vote(ctx context.Context, req *voter.Request) (*voter.Response, error) {
	p.updateLamportReq(req)
	log.Printf("Request received from %d L(%d, %d)\n", req.Id, p.id, p.clock)
	for {
		if p.state != HELD { //don't respond while in critical state
			break
		}
	}
	//if i dont want it or we both want it but they have a better Lamport (better = lower)
	if p.state == RELEASED || (p.clock > req.Clock || p.clock == req.Clock && p.id > req.Id){
		resp := &voter.Response{Id: p.id, Clock: p.clock}
		p.updateLamportResp(resp)
		resp = &voter.Response{Id: p.id, Clock: p.clock}
		log.Printf("Respond to %d L(%d, %d)\n", req.Id, resp.Id, resp.Clock)
		return resp, nil
	}
	//else if I have higher priority
	for  {
		if p.state == RELEASED { //don't respond until I'm done
			break
		}
	}
	resp := &voter.Response{Id: p.id, Clock: p.clock}
	p.updateLamportResp(resp)
	resp = &voter.Response{Id: p.id, Clock: p.clock}
	log.Printf("Respond to %d L(%d, %d)\n", req.Id, resp.Id, resp.Clock)
	return resp, nil
}

// try to enter critical state
func (p *peer) enter() {
	p.state = WANTED
	request := &voter.Request{Id: p.id, Clock: p.clock}

	for id, client := range p.clients {
		p.updateLamportReq(request)
		log.Printf("Request sent to: %d L(%d, %d)\n", id, p.id, p.clock)
		reply, err := client.Vote(p.ctx, request)
		if err != nil {
			log.Printf("%v", err)
		}
		p.updateLamportResp(reply)
		log.Printf("Reply recieved from %d L(%d, %d)\n", reply.Id, p.id, p.clock)
	}
	//after response from two peers
	p.state = HELD

	//Enter :-)
	doCriticalStuff()

	//Release Critical
	p.state = RELEASED
}

func doCriticalStuff() {
	log.Printf("Critical: I got permission :D")
}

func (p *peer) updateLamportReq(req *voter.Request) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.clock = max(p.clock, req.Clock) + 1
}

func (p *peer) updateLamportResp(resp *voter.Response) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.clock = max(p.clock, resp.Clock) + 1
}

func max(a int32, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
