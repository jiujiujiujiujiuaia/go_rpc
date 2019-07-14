package server

import (
	"go_rpc/codec"
	"go_rpc/transport"
	"sync"
)

type stubServer struct {
	codec      codec.Codec
	serviceMap sync.Map
	tr         transport.ServerTransport
	mutex      sync.Mutex
	shutdown   bool

	option Option
}