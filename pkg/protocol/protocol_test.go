package protocol

import (
	"bytes"
	"testing"
)

func TestVarIntRoundTrip(t *testing.T) {
	testCases := []int32{
		0, 1, 2, 127, 128, 255, 25565, 2097151, 2147483647, -1, -2147483648,
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		n, err := WriteVarInt(&buf, tc)
		if err != nil {
			t.Fatalf("WriteVarInt failed for %d: %v", tc, err)
		}
		if n != buf.Len() {
			t.Fatalf("byte count mismatch for %d: got %d, expected %d", tc, n, buf.Len())
		}

		decoded, readN, err := ReadVarInt(&buf)
		if err != nil {
			t.Fatalf("ReadVarInt failed for %d: %v", tc, err)
		}
		if decoded != tc {
			t.Fatalf("value mismatch: got %d, expected %d", decoded, tc)
		}
		if readN != n {
			t.Fatalf("bytes read mismatch: got %d, expected %d", readN, n)
		}
	}
}

func TestVarLongRoundTrip(t *testing.T) {
	testCases := []int64{
		0, 1, 127, 128, 255, 2147483647, 9223372036854775807, -1, -9223372036854775808,
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		n, err := WriteVarLong(&buf, tc)
		if err != nil {
			t.Fatalf("WriteVarLong failed for %d: %v", tc, err)
		}

		decoded, readN, err := ReadVarLong(&buf)
		if err != nil {
			t.Fatalf("ReadVarLong failed for %d: %v", tc, err)
		}
		if decoded != tc {
			t.Fatalf("value mismatch: got %d, expected %d", decoded, tc)
		}
		if readN != n {
			t.Fatalf("bytes read mismatch: got %d, expected %d", readN, n)
		}
	}
}

func TestPositionPacking(t *testing.T) {
	coords := [][3]int32{
		{0, 0, 0},
		{100, 64, -200},
		{-1500, -64, 3000},
		{30000000, 319, -30000000},
		{-30000000, -64, 30000000},
	}

	for _, c := range coords {
		var buf bytes.Buffer
		if err := WritePosition(&buf, c[0], c[1], c[2]); err != nil {
			t.Fatalf("WritePosition failed: %v", err)
		}
		x, y, z, err := ReadPosition(&buf)
		if err != nil {
			t.Fatalf("ReadPosition failed: %v", err)
		}
		if x != c[0] || y != c[1] || z != c[2] {
			t.Fatalf("coordinate mismatch: got (%d, %d, %d), expected (%d, %d, %d)", x, y, z, c[0], c[1], c[2])
		}
	}
}

func TestHandshakeRoundTrip(t *testing.T) {
	orig := &HandshakePacket{
		ProtocolVersion: 770,
		ServerAddress:   "127.0.0.1",
		ServerPort:      25565,
		NextState:       2,
	}

	var buf bytes.Buffer
	if err := SendPacket(&buf, orig); err != nil {
		t.Fatalf("SendPacket failed: %v", err)
	}

	raw, err := ReadRawPacket(&buf)
	if err != nil {
		t.Fatalf("ReadRawPacket failed: %v", err)
	}
	if raw.ID != 0x00 {
		t.Fatalf("unexpected packet ID: %d", raw.ID)
	}

	decoded := &HandshakePacket{}
	if err := decoded.Decode(bytes.NewReader(raw.Data)); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.ProtocolVersion != orig.ProtocolVersion ||
		decoded.ServerAddress != orig.ServerAddress ||
		decoded.ServerPort != orig.ServerPort ||
		decoded.NextState != orig.NextState {
		t.Fatalf("handshake mismatch: %+v vs %+v", decoded, orig)
	}
}

func TestPlayerPositionRoundTrip(t *testing.T) {
	orig := &PlayerPositionPacket{
		X:        128.5,
		Y:        72.0,
		Z:        -64.25,
		Yaw:      90.0,
		Pitch:    -15.5,
		OnGround: true,
	}

	var buf bytes.Buffer
	if err := SendPacket(&buf, orig); err != nil {
		t.Fatalf("SendPacket failed: %v", err)
	}

	raw, err := ReadRawPacket(&buf)
	if err != nil {
		t.Fatalf("ReadRawPacket failed: %v", err)
	}

	decoded := &PlayerPositionPacket{}
	if err := decoded.Decode(bytes.NewReader(raw.Data)); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.X != orig.X || decoded.Y != orig.Y || decoded.Z != orig.Z ||
		decoded.Yaw != orig.Yaw || decoded.Pitch != orig.Pitch || decoded.OnGround != orig.OnGround {
		t.Fatalf("player position mismatch: %+v vs %+v", decoded, orig)
	}
}

func TestBlockUpdateRoundTrip(t *testing.T) {
	orig := &BlockUpdatePacket{
		X:       -15,
		Y:       64,
		Z:       200,
		BlockID: 15, // redstone wire
	}

	var buf bytes.Buffer
	if err := SendPacket(&buf, orig); err != nil {
		t.Fatalf("SendPacket failed: %v", err)
	}

	raw, err := ReadRawPacket(&buf)
	if err != nil {
		t.Fatalf("ReadRawPacket failed: %v", err)
	}

	decoded := &BlockUpdatePacket{}
	if err := decoded.Decode(bytes.NewReader(raw.Data)); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.X != orig.X || decoded.Y != orig.Y || decoded.Z != orig.Z || decoded.BlockID != orig.BlockID {
		t.Fatalf("block update mismatch: %+v vs %+v", decoded, orig)
	}
}
