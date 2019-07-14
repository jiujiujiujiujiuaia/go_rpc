package transport

import (
	"io"
	"net"
)

type ServerTransport interface {
	Listen(network, addr string) error
	Accept() (ClientTransport, error)
	io.Closer
}

type ServerSocket struct {
	ln net.Listener
}

func NewServerTransport()ServerTransport{
	return new(ServerSocket)
}

func (s *ServerSocket) Listen(network, addr string) error {
	ln, err := net.Listen(network, addr)
	s.ln = ln
	return err
}

func (s *ServerSocket) Accept() (ClientTransport, error) {
	conn, err := s.ln.Accept()
	return &ClientSocket{conn: conn}, err
}

func (s *ServerSocket) Close() error {
	return s.ln.Close()
}
