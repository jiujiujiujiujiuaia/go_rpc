package transport

import (
	"io"
	"net"
)

type ClientTransport interface {
	Dial(network, addr string) error
	io.ReadWriteCloser
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type ClientSocket struct {
	conn net.Conn
}

func NewClientTransport()  ClientTransport{
	return new(ClientSocket)
}

func (s *ClientSocket) Dial(network,addr string)error{
	conn,err := net.Dial(network,addr)
	s.conn = conn
	return err
}

func (s *ClientSocket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s ClientSocket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *ClientSocket) Read(p []byte) (n int, err error) {
	return s.conn.Read(p)
}

func (s *ClientSocket) Write(p []byte) (n int, err error) {
	return s.conn.Write(p)
}

func (s *ClientSocket) Close() error {
	return s.conn.Close()
}