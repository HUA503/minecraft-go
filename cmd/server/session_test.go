package main

import (
	"net"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/protocol"
)

func TestClientSessionDisconnectsWhenOutboundQueueIsFull(t *testing.T) {
	serverConn, peerConn := net.Pipe()
	defer peerConn.Close()

	sess := &ClientSession{
		ID:       1,
		conn:     serverConn,
		server:   &Server{},
		outbound: make(chan protocol.Packet, 1),
		done:     make(chan struct{}),
	}
	packet := &protocol.PingPacket{Payload: 1}
	if err := sess.Send(packet); err != nil {
		t.Fatalf("first queued packet failed: %v", err)
	}
	if err := sess.Send(packet); err == nil {
		t.Fatal("expected a full outbound queue to reject the packet")
	}

	select {
	case <-sess.done:
	case <-time.After(time.Second):
		t.Fatal("slow session was not disconnected after queue overflow")
	}
	if _, err := peerConn.Write([]byte{1}); err == nil {
		t.Fatal("peer connection remained open after queue overflow")
	}
}
