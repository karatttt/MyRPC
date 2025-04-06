package codec

import (
	"context"
	"encoding/json"
	"encoding/binary"
	"fmt"
	"MyRPC/core/internel"
	"bufio"
	"io"
	"net"
)

var DefaultClientCodec = NewClientCodec()

var DefaultServerCodec = NewServerCodec()

// 定义帧格式常量
const (
	// 魔数，用于标识协议
	MagicNumber = 0x1234
	// 版本号
	Version = 1
	// 帧头长度（魔数2字节 + 版本1字节 + 消息类型1字节 + 序列号4字节 + 消息体长度4字节）
	HeaderLength = 12
)

// 消息类型
const (
	MessageTypeRequest  = 1
	MessageTypeResponse = 2
)

// 帧头结构
type FrameHeader struct {
	MagicNumber uint16 // 魔数
	Version     uint8  // 版本号
	MessageType uint8  // 消息类型
	SequenceID  uint32 // 序列号
	BodyLength  uint32 // 消息体长度
}

type clientcodec struct{}

type servercodec struct{}

type Codec interface {
	Encode(ctx context.Context, reqData []byte) ([]byte, error)
	Decode(ctx context.Context, frame []byte) ([]byte, error)
}

func NewClientCodec() Codec {
	return &clientcodec{}
}

func NewServerCodec() Codec {
	return &servercodec{}
}

// ------CLIENT
func (c *clientcodec) Encode(ctx context.Context, reqData []byte) ([]byte, error) {
	header := FrameHeader{
		MagicNumber: MagicNumber,
		Version:     Version,
		MessageType: MessageTypeRequest,
		SequenceID:  1, // 这里应该使用递增的序列号
		BodyLength:  uint32(len(reqData)),
	}

	// 将帧头写入缓冲区
	headerBuf := make([]byte, HeaderLength)
	binary.BigEndian.PutUint16(headerBuf[0:], header.MagicNumber)
	headerBuf[2] = header.Version
	headerBuf[3] = header.MessageType
	binary.BigEndian.PutUint32(headerBuf[4:], header.SequenceID)
	binary.BigEndian.PutUint32(headerBuf[8:], header.BodyLength)

	// 创建帧
	frame := make([]byte, 0)
	// 组装帧头
	frame = append(frame, headerBuf...)
	// 组装请求协议
	protodata, err := createProtocolData(ctx)
	if err != nil {
		return nil, fmt.Errorf("create protocol data error: %v", err)
	}
	frame = append(frame, protodata...)
	// 组装帧体
	frame = append(frame, reqData...)
	return frame, nil
}

func createProtocolData(ctx context.Context) ([]byte, error) {
	// 创建协议数据
	protocolData := make([]byte, 0)
	// 组装协议数据,从context中获取msg，并将msg中的属性添加到ptotocolData中
	msg := internel.GetMessage(ctx)
	protocolData = append(protocolData, msg.GetServiceName()...)
	protocolData = append(protocolData, msg.GetMethodName()...)
	return protocolData, nil
}

func (c *clientcodec) Decode(ctx context.Context, frame []byte) ([]byte, error) {
	// 获取帧里面的数据体
	data := frame[HeaderLength:]
	return data, nil
}


// ------SERVER
func (c *servercodec) Encode(ctx context.Context, reqData []byte) ([]byte, error) {
	return json.Marshal(reqData)
}

func (c *servercodec) Decode(ctx context.Context, frame []byte) ([]byte, error) {
	// 获取帧里面的数据体
	data := frame[HeaderLength:]

	// 将帧的请求协议的数据存到context的msg中
	msg := internel.GetMessage(ctx)
	msg.WithMethodName(string(data[0:]))
	msg.WithServiceName(string(data[1:]))
	return data, nil
}


// ------COMMON
func ReadFrame(conn net.Conn) ([]byte, error) {
	// 读取TCP帧并存到缓冲区
	buf := bufio.NewReader(conn)

	// 读取帧头
	headerBuf := make([]byte, HeaderLength)
	_, err := io.ReadFull(buf, headerBuf)
	if err != nil {
		return nil, fmt.Errorf("read header error: %v", err)
	}

	// 解析帧头
	header := FrameHeader{
		MagicNumber: binary.BigEndian.Uint16(headerBuf[0:]),
		Version:     headerBuf[2],
		MessageType: headerBuf[3],
		SequenceID:  binary.BigEndian.Uint32(headerBuf[4:]),
		BodyLength:  binary.BigEndian.Uint32(headerBuf[8:]),
	}

	// 验证魔数和版本
	if header.MagicNumber != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: %d", header.MagicNumber)
	}
	if header.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// 读取帧体
	frameBody := make([]byte, header.BodyLength)
	_, err = io.ReadFull(buf, frameBody)
	if err != nil {
		return nil, fmt.Errorf("read body error: %v", err)
	}

	return frameBody, nil
}
