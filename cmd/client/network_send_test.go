package main

import (
	"net"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/protocol"
)

func TestClientDisconnectsWhenOutboundQueueIsFull(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	clientConn, peerConn := net.Pipe()
	defer peerConn.Close()

	c := &Client{
		conn:             clientConn,
		networkSendQueue: make(chan protocol.Packet, 1),
		networkSendStop:  make(chan struct{}),
	}
	packet := &protocol.PingPacket{Payload: 1}
	if err := c.sendPacket(packet); err != nil {
		t.Fatalf("first queued packet failed: %v", err)
	}
	if err := c.sendPacket(packet); err == nil {
		t.Fatal("expected a full outbound queue to reject the packet")
	}
	if !c.networkDisconnected.Load() {
		t.Fatal("queue overflow did not mark the connection disconnected")
	}
	select {
	case <-c.networkSendStop:
	default:
		t.Fatal("queue overflow did not stop the network sender")
	}
	if _, err := peerConn.Write([]byte{1}); err == nil {
		t.Fatal("peer connection remained open after queue overflow")
	}
}

