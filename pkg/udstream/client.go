// pkg/udstream/client.go
package udstream

import (
	"context"
	"net"
	"sync"

	"github.com/artemstarovojtovigorevich-eng/go-udstream/proto/pb/proto"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
	srcID  uint32
	mutex  sync.Mutex
}

// NewClient создаёт UDP‑клиент для твоего протокола.
func NewClient(dstAddr string, srcID uint32) (*Client, error) {
	remote, err := net.ResolveUDPAddr("udp", dstAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		remote: remote,
		srcID:  srcID,
		mutex:  sync.Mutex{},
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// send отправляет сериализованный протобуф‑пакет.
func (c *Client) send(packet *pb.Packet) error {
	data, err := proto.Marshal(packet)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(data)
	return err
}

// SendDelta отправляет группу изменений (для OPC‑данных).
func (c *Client) SendDelta(ctx context.Context, seq uint32, msgs []*pb.Message) error {
	batch := &pb.DeltaBatch{
		Seq:       seq,
		Messages:  msgs,
	}

	packet := &pb.Packet{
		Payload: &pb.Packet_Delta{Delta: batch},
	}

	return c.send(packet)
}

// SendFull отправляет полный снапшот OPC UA‑модели.
func (c *Client) SendFull(ctx context.Context, timestamp uint64, nodes []*pb.Message) error {
	full := &pb.FullSnapshot{
		Timestamp: timestamp,
		SourceId:  c.srcID,
		Nodes:     nodes,
	}

	packet := &pb.Packet{
		Payload: &pb.Packet_Full{Full: full},
	}

	return c.send(packet)
}

// Addr возвращает адрес сервера.
func (c *Client) Addr() *net.UDPAddr {
	return c.remote
}

// SrcID возвращает ID источника.
func (c *Client) SrcID() uint32 {
	return c.srcID
}