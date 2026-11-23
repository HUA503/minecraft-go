package main

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/render"
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

func TestConnectClosesSocketWhenLoginResponseIsInvalid(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		if _, err := protocol.ReadRawPacket(conn); err != nil { // handshake
			serverDone <- err
			return
		}
		if _, err := protocol.ReadRawPacket(conn); err != nil { // login start
			serverDone <- err
			return
		}
		if err := protocol.WriteRawPacket(conn, 0x02, []byte{1}); err != nil {
			serverDone <- err
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, err = protocol.ReadRawPacket(conn)
		if err == nil {
			serverDone <- io.ErrUnexpectedEOF
			return
		}
		serverDone <- nil
	}()

	client := &Client{
		serverAddr: listener.Addr().String(),
		username:   "test",
		camera:     &render.Camera{},
	}
	if err := client.Connect(); err == nil {
		t.Fatal("expected malformed login response to fail")
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("server did not observe failed connection cleanup: %v", err)
	}
}

