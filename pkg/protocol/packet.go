package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ConnectionState represents the Minecraft client connection phase.
type ConnectionState int

// ProtocolVersion is the canonical protocol version for Minecraft Java Edition 26.3.
const ProtocolVersion int32 = 777

const (
	StateHandshake ConnectionState = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
)

func (s ConnectionState) String() string {
	switch s {
	case StateHandshake:
		return "Handshake"
	case StateStatus:
		return "Status"
	case StateLogin:
		return "Login"
	case StateConfiguration:
		return "Configuration"
	case StatePlay:
		return "Play"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// Packet defines the generic interface for serializing and deserializing Minecraft packets.
type Packet interface {
	PacketID() int32
	Encode(w io.Writer) error
	Decode(r io.Reader) error
}

// PacketHeader describes a raw framed packet.
type RawPacket struct {
	ID   int32
	Data []byte
}

func (r *RawPacket) Reader() io.Reader {
	return bytes.NewReader(r.Data)
}

// ReadRawPacket reads a length-prefixed packet frame from an io.Reader.
func ReadRawPacket(r io.Reader) (*RawPacket, error) {
	length, _, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	if length <= 0 {
		return nil, fmt.Errorf("invalid packet length: %d", length)
	}

	frame := make([]byte, length)
	if _, err := io.ReadFull(r, frame); err != nil {
		return nil, err
	}

	buf := bytes.NewReader(frame)
	packetID, idLen, err := ReadVarInt(buf)
	if err != nil {
		return nil, err
	}

	payload := frame[idLen:]
	return &RawPacket{
		ID:   packetID,
		Data: payload,
	}, nil
}

// WriteRawPacket writes a length-prefixed packet frame to an io.Writer.
func WriteRawPacket(w io.Writer, packetID int32, data []byte) error {
	var idBuf bytes.Buffer
	if _, err := WriteVarInt(&idBuf, packetID); err != nil {
		return err
	}

	totalLen := int32(idBuf.Len() + len(data))
	if _, err := WriteVarInt(w, totalLen); err != nil {
		return err
	}
	if _, err := w.Write(idBuf.Bytes()); err != nil {
		return err
	}
	if len(data) > 0 {
		_, err := w.Write(data)
		return err
	}
	return nil
}

// SendPacket serializes a typed Packet into a length-prefixed frame.
func SendPacket(w io.Writer, p Packet) error {
	var buf bytes.Buffer
	if err := p.Encode(&buf); err != nil {
		return err
	}
	return WriteRawPacket(w, p.PacketID(), buf.Bytes())
}

// --- Specific Packets ---

// HandshakePacket (Serverbound 0x00 in Handshake state)
type HandshakePacket struct {
	ProtocolVersion int32
	ServerAddress   string
	ServerPort      uint16
	NextState       int32 // 1: Status, 2: Login
}

func (p *HandshakePacket) PacketID() int32 { return 0x00 }
func (p *HandshakePacket) Encode(w io.Writer) error {
	if _, err := WriteVarInt(w, p.ProtocolVersion); err != nil {
		return err
	}
	if err := WriteString(w, p.ServerAddress); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, p.ServerPort); err != nil {
		return err
	}
	_, err := WriteVarInt(w, p.NextState)
	return err
}

func (p *HandshakePacket) Decode(r io.Reader) error {
	var err error
	if p.ProtocolVersion, _, err = ReadVarInt(r); err != nil {
		return err
	}
	if p.ServerAddress, err = ReadString(r, 255); err != nil {
		return err
	}
	if err = binary.Read(r, binary.BigEndian, &p.ServerPort); err != nil {
		return err
	}
	p.NextState, _, err = ReadVarInt(r)
	return err
}

// StatusRequestPacket (Serverbound 0x00 in Status state)
type StatusRequestPacket struct{}

func (p *StatusRequestPacket) PacketID() int32        { return 0x00 }
func (p *StatusRequestPacket) Encode(w io.Writer) error { return nil }
func (p *StatusRequestPacket) Decode(r io.Reader) error { return nil }

// StatusResponsePacket (Clientbound 0x00 in Status state)
type StatusResponsePacket struct {
	JSONResponse string
}

func (p *StatusResponsePacket) PacketID() int32 { return 0x00 }
func (p *StatusResponsePacket) Encode(w io.Writer) error {
	return WriteString(w, p.JSONResponse)
}
func (p *StatusResponsePacket) Decode(r io.Reader) error {
	var err error
	p.JSONResponse, err = ReadString(r, 32767)
	return err
}

// PingPacket (Bidirectional 0x01 in Status state)
type PingPacket struct {
	Payload int64
}

func (p *PingPacket) PacketID() int32 { return 0x01 }
func (p *PingPacket) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, p.Payload)
}
func (p *PingPacket) Decode(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &p.Payload)
}

// LoginStartPacket (Serverbound 0x00 in Login state)
type LoginStartPacket struct {
	Username string
	PlayerUUID [16]byte
}

func (p *LoginStartPacket) PacketID() int32 { return 0x00 }
func (p *LoginStartPacket) Encode(w io.Writer) error {
	if err := WriteString(w, p.Username); err != nil {
		return err
	}
	_, err := w.Write(p.PlayerUUID[:])
	return err
}
func (p *LoginStartPacket) Decode(r io.Reader) error {
	var err error
	if p.Username, err = ReadString(r, 16); err != nil {
		return err
	}
	_, err = io.ReadFull(r, p.PlayerUUID[:])
	return err
}

// LoginSuccessPacket (Clientbound 0x02 in Login state)
type LoginSuccessPacket struct {
	UUID     [16]byte
	Username string
}

func (p *LoginSuccessPacket) PacketID() int32 { return 0x02 }
func (p *LoginSuccessPacket) Encode(w io.Writer) error {
	if _, err := w.Write(p.UUID[:]); err != nil {
		return err
	}
	if err := WriteString(w, p.Username); err != nil {
		return err
	}
	// Properties count = 0 (VarInt)
	_, err := WriteVarInt(w, 0)
	return err
}
func (p *LoginSuccessPacket) Decode(r io.Reader) error {
	if _, err := io.ReadFull(r, p.UUID[:]); err != nil {
		return err
	}
	var err error
	if p.Username, err = ReadString(r, 16); err != nil {
		return err
	}
	propCount, _, err := ReadVarInt(r)
	if err != nil {
		return err
	}
	if propCount > 0 {
		// skip properties for minimal flow
		for i := int32(0); i < propCount; i++ {
			if _, err := ReadString(r, 255); err != nil {
				return err
			}
			if _, err := ReadString(r, 32767); err != nil {
				return err
			}
			var hasSig byte
			if err := binary.Read(r, binary.BigEndian, &hasSig); err != nil {
				return err
			}
			if hasSig != 0 {
				if _, err := ReadString(r, 32767); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// FinishConfigurationPacket (Bidirectional in modern Configuration state)
type FinishConfigurationPacket struct{}

func (p *FinishConfigurationPacket) PacketID() int32        { return 0x03 }
func (p *FinishConfigurationPacket) Encode(w io.Writer) error { return nil }
func (p *FinishConfigurationPacket) Decode(r io.Reader) error { return nil }

// PlayerPositionPacket (Clientbound/Serverbound in Play state)
type PlayerPositionPacket struct {
	X     float64
	Y     float64
	Z     float64
	Yaw   float32
	Pitch float32
	OnGround bool
}

func (p *PlayerPositionPacket) PacketID() int32 { return 0x1A }
func (p *PlayerPositionPacket) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, p.X); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, p.Y); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, p.Z); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, p.Yaw); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, p.Pitch); err != nil {
		return err
	}
	var og byte
	if p.OnGround {
		og = 1
	}
	return binary.Write(w, binary.BigEndian, og)
}

func (p *PlayerPositionPacket) Decode(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &p.X); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &p.Y); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &p.Z); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &p.Yaw); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &p.Pitch); err != nil {
		return err
	}
	var og byte
	if err := binary.Read(r, binary.BigEndian, &og); err != nil {
		return err
	}
	p.OnGround = (og != 0)
	return nil
}

// BlockUpdatePacket (Clientbound in Play state)
type BlockUpdatePacket struct {
	X       int32
	Y       int32
	Z       int32
	BlockID int32
}

func (p *BlockUpdatePacket) PacketID() int32 { return 0x09 }
func (p *BlockUpdatePacket) Encode(w io.Writer) error {
	if err := WritePosition(w, p.X, p.Y, p.Z); err != nil {
		return err
	}
	_, err := WriteVarInt(w, p.BlockID)
	return err
}

func (p *BlockUpdatePacket) Decode(r io.Reader) error {
	var err error
	p.X, p.Y, p.Z, err = ReadPosition(r)
	if err != nil {
		return err
	}
	p.BlockID, _, err = ReadVarInt(r)
	return err
}

// KeepAlivePacket (Bidirectional in Play state)
type KeepAlivePacket struct {
	ID int64
}

func (p *KeepAlivePacket) PacketID() int32 { return 0x26 }
func (p *KeepAlivePacket) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, p.ID)
}
func (p *KeepAlivePacket) Decode(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &p.ID)
}

// DisconnectPacket (Clientbound in Play / Login state)
type DisconnectPacket struct {
	Reason string
}

func (p *DisconnectPacket) PacketID() int32 { return 0x1D }
func (p *DisconnectPacket) Encode(w io.Writer) error {
	return WriteString(w, p.Reason)
}
func (p *DisconnectPacket) Decode(r io.Reader) error {
	var err error
	p.Reason, err = ReadString(r, 32767)
	return err
}
