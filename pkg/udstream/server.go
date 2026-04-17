// pkg/udstream/server.go
package udstream

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/artemstarovojtovigorevich-eng/go-udstream/proto/pb/proto"
	"google.golang.org/protobuf/proto"
)

type Handler interface {
	OnDelta(*pb.DeltaBatch)
	OnFull(*pb.FullSnapshot)
}

type Server struct {
	addr    *net.UDPAddr
	handler Handler
	running bool
	conn    *net.UDPConn
	wg      sync.WaitGroup
	mu      sync.Mutex
}

type Config struct {
	Addr string
}

func NewServer(cfg *Config, handler Handler) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp", cfg.Addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		addr:    addr,
		handler: handler,
		conn:    conn,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run(ctx)

	<-ctx.Done()
	s.conn.Close()
	s.wg.Wait()
	return ctx.Err()
}

func (s *Server) run(ctx context.Context) {
	defer s.wg.Done()

	buffer := make([]byte, 65507) // max UDP payload

	for {
		n, _, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				// логируешь ошибку, но не падаешь
				continue
			}
		}

		packet := &pb.Packet{}
		if err := proto.Unmarshal(buffer[:n], packet); err != nil {
			// log.Printf("unmarshal error from %s: %v", addr, err)
			continue
		}

		switch x := packet.Payload.(type) {
		case *pb.Packet_Delta:
			s.handler.OnDelta(x.Delta)

		case *pb.Packet_Full:
			s.handler.OnFull(x.Full)

		default:
			// log.Printf("unknown packet type from %s", addr)
		}
	}
}

func (s *Server) Addr() *net.UDPAddr {
	return s.addr
}