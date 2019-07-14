package common_type

import "context"

type RPCServer interface {
	//注册服务实例，rcvr是receiver的意思，它是我们对外暴露的方法的实现者，metaData是注册服务时携带的额外的元数据，它描述了rcvr的其他信息
	Register(rcvr interface{}, metaData map[string]string) error
	//开始对外提供服务
	Serve(network string, addr string) error
}
type RPCClient interface {
	//Go表示异步调用
	Go(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}, done chan *Call) *Call
	//Call表示异步调用
	Call(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}) error
//	Close() error
}

//调用结构体，主要用来记录调用信息以及是否调用完成
type Call struct {
	ServiceMethod string      // 服务名.方法名
	Args          interface{} // 参数
	Reply         interface{} // 返回值（指针类型）
	Error         error       // 错误信息
	Done          chan *Call  // 在调用结束时激活
}

func (c *Call) Dones() {
	c.Done <- c
}

