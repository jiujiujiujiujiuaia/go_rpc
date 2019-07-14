package client

import (
	"context"
	"fmt"
	"github.com/lexkong/log"
	"github.com/pkg/errors"
	"go_rpc/codec"
	"go_rpc/common_type"
	"go_rpc/protocol"
	"go_rpc/transport"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//客户端stub
type clientStub struct {
	seq uint64 //序列号
	rwc io.ReadWriteCloser  //具有读写关闭功能的channel
	codec codec.Codec  //编解码

	pendingCalls sync.Map
	mutex        sync.Mutex
	shutdown     bool
	option       Option

}

func NewClientStub(network,addr string)(common_type.RPCClient,error){
	clientStub := new(clientStub)

	clientSocket := transport.NewClientTransport()
	err := clientSocket.Dial(network,addr)
	if err != nil {
		return nil,err
	}
	clientStub.codec = new(codec.MessagePackCodec)
	clientStub.rwc = clientSocket

	//开启回包接收
	go clientStub.input()
	return clientStub,nil
}

//发起clientStub调用
func (stub *clientStub) Call(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}) error{
	seq := atomic.AddUint64(&stub.seq,1)
	ctx = context.WithValue(ctx,protocol.RequestSeqKey,seq)

	canFn := func() {}
	//如果客户端设置了请求超时时间，就衍生一个子超时ctx
	if stub.option.RequestTimeout != time.Duration(0) {
		ctx, canFn = context.WithTimeout(ctx, stub.option.RequestTimeout)
		metaDataInterface := ctx.Value(protocol.MetaDataKey)
		var metaData map[string]string
		if metaDataInterface == nil {
			metaData = make(map[string]string)
		} else {
			metaData = metaDataInterface.(map[string]string)
		}
		metaData[protocol.RequestTimeoutKey] = stub.option.RequestTimeout.String()
		ctx = context.WithValue(ctx, protocol.MetaDataKey, metaData)
	}

	done := make(chan *common_type.Call, 1)
	call := stub.Go(ctx,serviceMethod,arg,reply,done)
	select {
		//如果子ctx.Done函数触发了，说明父ctx超时结束了
		case <- ctx.Done():
			call.Error = errors.New("超时了")
		case <-call.Done:
			fmt.Println("请求完成")
	}
	return call.Error
}

//拼装一下，最后把数据序列化后发送到socket缓冲区
func (stub *clientStub) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *common_type.Call) *common_type.Call {
	call := new(common_type.Call)
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = reply
	done = make(chan *common_type.Call,10)
	call.Done  = done
	stub.send(ctx,call)
	return call
}


//拼接request，然后序列化，然后写入到socket中
func (stub *clientStub) send(ctx context.Context,call *common_type.Call){
	seq := ctx.Value(protocol.RequestSeqKey).(uint64)

	request := protocol.NewMessgae()
	request.Seq = seq
	request.MessageType = 0
	serviceAndMehod := strings.Split(call.ServiceMethod,".")
	request.ServiceName = serviceAndMehod[0]
	request.MethodName = serviceAndMehod[1]
	request.SerializeType = 0

	//序列化参数（序列化算法有二进制，有文本的，二进制就是pb，文本的就是json）
	requestData,err := stub.codec.Encode(call.Args)
	log.Infof("序列化请求body=[%+v]",requestData)
	if err != nil {
		log.Infof("client encode args error : [%+v]",err)
		//发生错误，请求快速返回并记录错误
		call.Error = err
		call.Dones()
		return
	}
	request.Data = requestData
	//拼装request，按照协议规定的，0-3字节为是魔数之类的拼装协议，以字节为单位
	data := protocol.EncodeMessage(request)
	log.Infof("完整序列化后的完成的包(魔数+版本+请求头+请求体)=[%+v]",data)
	//写入到socker缓冲区
	_,err = stub.rwc.Write(data)
	//
	if err != nil {
		log.Infof("client encode request error : [%+v]",err)
		//发生错误，请求快速返回并记录错误
		call.Error = err
		call.Dones()
		return
	}
}

//处理从服务端发回的包并解析
func (stub *clientStub)input(){
	var err error
	var response *protocol.Message
	for err == nil {
		//解码
		response, err = protocol.DecodeMessage(stub.rwc)
		if err != nil {
			break
		}

		seq := response.Seq
		callInterface, _ := stub.pendingCalls.Load(seq)
		call := callInterface.(*common_type.Call)
		stub.pendingCalls.Delete(seq)

		switch {
		case call == nil:
			//请求已经被清理掉了，可能是已经超时了
		case response.Error != "":
			call.Error = errors.New(response.Error)
			call.Dones()
		default:
			err = stub.codec.Decode(response.Data, call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.Dones()
			log.Infof("请求成功收到回包")
		}
	}
}