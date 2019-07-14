package protocol

import (
	"encoding/binary"
	"errors"
	"github.com/vmihailenco/msgpack"
	"io"
)

const (
	RequestSeqKey     = "rpc_request_seq"
	RequestTimeoutKey = "rpc_request_timeout"
	MetaDataKey       = "rpc_meta_data"
)

type Message struct {
	*Header
	Data []byte
}

type Header struct {
	Seq uint64 //序号, 用来唯一标识请求或响应
	MessageType byte //消息类型，用来标识一个消息是请求还是响应
	CompressType byte //压缩类型，用来标识一个消息的压缩方式
	SerializeType byte //序列化类型，用来标识消息体采用的编码方式
	StatusCode byte //状态类型，用来标识一个请求是正常还是异常
	ServiceName string //服务名
	MethodName string  //方法名
	Error string //方法调用发生的异常
	MetaData map[string]string //其他元数据
}

func NewMessgae() *Message{
	return &Message{Header:&Header{}}
}


//-------------------------------------------------------------------------------------------------
//|2byte|1byte  |4byte       |4byte        | header length |(total length - header length - 4byte)|
//-------------------------------------------------------------------------------------------------
//|magic|version|total length|header length|     header    |                    body              |
//-------------------------------------------------------------------------------------------------
func EncodeMessage(msg *Message)[] byte{
	//一个字节就是8个bit，两位十六进制
	first3bytes := []byte{0xab, 0xba, 0x00}
	headerBytes, _ := msgpack.Marshal(msg.Header)

	//totolLen = headerLen + bodyLen + header_length(4)
	totalLen := 4 + len(headerBytes) + len(msg.Data)
	totalLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(totalLenBytes, uint32(totalLen))

	data := make([]byte, totalLen+7)
	start := 0
	copyFullWithOffset(data, first3bytes, &start)
	copyFullWithOffset(data, totalLenBytes, &start)

	headerLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLenBytes, uint32(len(headerBytes)))
	copyFullWithOffset(data, headerLenBytes, &start)
	copyFullWithOffset(data, headerBytes, &start)
	copyFullWithOffset(data, msg.Data, &start)

	return data
}

func DecodeMessage(r io.Reader) (msg *Message, err error) {
	first3bytes := make([]byte, 3)
	//阻塞读
	_, err = io.ReadFull(r, first3bytes)
	if err != nil {
		return
	}
	//校验前三位是否是魔数等信息
	if !checkMagic(first3bytes[:2]) {
		err = errors.New("wrong protocol")
		return
	}
	totalLenBytes := make([]byte, 4)
	_, err = io.ReadFull(r, totalLenBytes)
	if err != nil {
		return
	}
	totalLen := int(binary.BigEndian.Uint32(totalLenBytes))
	//请求包总长小于32个比特
	if totalLen < 4 {
		err = errors.New("invalid total length")
		return
	}
	data := make([]byte, totalLen)
	_, err = io.ReadFull(r, data)
	headerLen := int(binary.BigEndian.Uint32(data[:4]))
	headerBytes := data[4 : headerLen+4]
	header := &Header{}
	err = msgpack.Unmarshal(headerBytes, header)
	if err != nil {
		return
	}
	msg = new(Message)
	msg.Header = header
	msg.Data = data[headerLen+4:]
	return

}

func checkMagic(bytes []byte) bool {
	return bytes[0] == 0xab && bytes[1] == 0xba
}

func copyFullWithOffset(dst []byte, src []byte, start *int) {
	copy(dst[*start:*start+len(src)], src)
	*start = *start + len(src)
}

