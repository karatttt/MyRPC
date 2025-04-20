package codec

import (
	"MyRPC/core/internel"
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
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
	// 帧头长度（魔数2字节 + 版本1字节 + 消息类型1字节 + 序列号4字节 + 协议数据长度4字节 + 消息体长度4字节）
	HeaderLength = 16
)

// 消息类型
const (
	MessageTypeRequest  = 1
	MessageTypeResponse = 2
)

// 帧头结构
type FrameHeader struct {
	MagicNumber    uint16 // 魔数
	Version        uint8  // 版本号
	MessageType    uint8  // 消息类型
	SequenceID     uint32 // 序列号
	ProtocolLength uint32 // 协议数据长度
	BodyLength     uint32 // 消息体长度
}

type clientcodec struct{}

type servercodec struct{}

type Codec interface {
	Encode(ctx context.Context, reqData []byte) ([]byte, error)
	Decode(msg internel.Message, frame []byte) ([]byte, error)
}

func NewClientCodec() Codec {
	return &clientcodec{}
}

func NewServerCodec() Codec {
	return &servercodec{}
}

// ------CLIENT
func (c *clientcodec) Encode(ctx context.Context, reqData []byte) ([]byte, error) {
	// 获取服务名和方法名
	ctx, msg := internel.GetMessage(ctx)

	// 创建协议数据
	protocolData := &ProtocolData{
		ServiceName: msg.GetServiceName(),
		MethodName:  msg.GetMethodName(),
	}

	// 序列化协议数据
	protocolDataBytes, err := protocolData.Serialize()
	if err != nil {
		return nil, fmt.Errorf("serialize protocol data error: %v", err)
	}

	// 创建帧头
	header := FrameHeader{
		MagicNumber:    MagicNumber,
		Version:        Version,
		MessageType:    MessageTypeRequest,
		SequenceID:     1, // 这里应该使用递增的序列号
		ProtocolLength: uint32(len(protocolDataBytes)),
		BodyLength:     uint32(len(reqData)),
	}

	// 将帧头写入缓冲区
	headerBuf := make([]byte, HeaderLength)
	binary.BigEndian.PutUint16(headerBuf[0:], header.MagicNumber)
	headerBuf[2] = header.Version
	headerBuf[3] = header.MessageType
	binary.BigEndian.PutUint32(headerBuf[4:], header.SequenceID)
	binary.BigEndian.PutUint32(headerBuf[8:], header.ProtocolLength)
	binary.BigEndian.PutUint32(headerBuf[12:], header.BodyLength)

	// 组装完整帧
	frame := make([]byte, 0)
	frame = append(frame, headerBuf...)
	frame = append(frame, protocolDataBytes...)
	frame = append(frame, reqData...)

	return frame, nil
}

func (c *clientcodec) Decode(msg internel.Message, frame []byte) ([]byte, error) {
	// 解析帧头
	header := FrameHeader{
		MagicNumber:    binary.BigEndian.Uint16(frame[0:]),
		Version:        frame[2],
		MessageType:    frame[3],
		SequenceID:     binary.BigEndian.Uint32(frame[4:]),
		ProtocolLength: binary.BigEndian.Uint32(frame[8:]),
		BodyLength:     binary.BigEndian.Uint32(frame[12:]),
	}

	// 验证魔数和版本
	if header.MagicNumber != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: %d", header.MagicNumber)
	}
	if header.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// 返回消息体
	return frame[HeaderLength+header.ProtocolLength:], nil
}

// ------SERVER
func (c *servercodec) Encode(ctx context.Context, reqData []byte) ([]byte, error) {
	// 获取服务名和方法名
	ctx, msg := internel.GetMessage(ctx)

	// 创建协议数据
	protocolData := &ProtocolData{
		ServiceName: msg.GetServiceName(),
		MethodName:  msg.GetMethodName(),
	}

	// 序列化协议数据
	protocolDataBytes, err := protocolData.Serialize()
	if err != nil {
		return nil, fmt.Errorf("serialize protocol data error: %v", err)
	}

	// 创建帧头
	header := FrameHeader{
		MagicNumber:    MagicNumber,
		Version:        Version,
		MessageType:    MessageTypeRequest,
		SequenceID:     1, // 这里应该使用递增的序列号
		ProtocolLength: uint32(len(protocolDataBytes)),
		BodyLength:     uint32(len(reqData)),
	}

	// 将帧头写入缓冲区
	headerBuf := make([]byte, HeaderLength)
	binary.BigEndian.PutUint16(headerBuf[0:], header.MagicNumber)
	headerBuf[2] = header.Version
	headerBuf[3] = header.MessageType
	binary.BigEndian.PutUint32(headerBuf[4:], header.SequenceID)
	binary.BigEndian.PutUint32(headerBuf[8:], header.ProtocolLength)
	binary.BigEndian.PutUint32(headerBuf[12:], header.BodyLength)

	// 组装完整帧
	frame := make([]byte, 0)
	frame = append(frame, headerBuf...)
	frame = append(frame, protocolDataBytes...)
	frame = append(frame, reqData...)

	return frame, nil
}

func (c *servercodec) Decode(msg internel.Message, frame []byte) ([]byte, error) {
	// 解析帧头
	header := FrameHeader{
		MagicNumber:    binary.BigEndian.Uint16(frame[0:]),
		Version:        frame[2],
		MessageType:    frame[3],
		SequenceID:     binary.BigEndian.Uint32(frame[4:]),
		ProtocolLength: binary.BigEndian.Uint32(frame[8:]),
		BodyLength:     binary.BigEndian.Uint32(frame[12:]),
	}

	// 验证魔数和版本
	if header.MagicNumber != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: %d", header.MagicNumber)
	}
	if header.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// 提取协议数据
	protocolData := frame[HeaderLength : HeaderLength+header.ProtocolLength]

	// 解析协议数据
	proto, err := DeserializeProtocolData(protocolData)
	if err != nil {
		return nil, fmt.Errorf("parse protocol data error: %v", err)
	}

	// 设置到消息中
	msg.WithServiceName(proto.ServiceName)
	msg.WithMethodName(proto.MethodName)

	// 返回消息体
	return frame[HeaderLength+header.ProtocolLength:], nil
}

// ------COMMON
func ReadFrame(conn net.Conn) ([]byte, error) {
	buf := bufio.NewReader(conn)

	// 读取帧头
	headerBuf := make([]byte, HeaderLength)
	n, err := io.ReadFull(buf, headerBuf)
	if err != nil {
		if err == io.EOF {
			fmt.Printf("客户端没有发送任何数据，连接就关闭了")
			return nil, err
		} else if err == io.ErrUnexpectedEOF {
			fmt.Printf("收到部分数据 %d 字节，连接就断开了", n)
			return nil, err
		} else {
			fmt.Printf("读取异常，err: %v", err)
			return nil, err
		}
	}

	// 正确解析所有字段
	header := FrameHeader{
		MagicNumber:    binary.BigEndian.Uint16(headerBuf[0:2]),
		Version:        headerBuf[2],
		MessageType:    headerBuf[3],
		SequenceID:     binary.BigEndian.Uint32(headerBuf[4:8]),
		ProtocolLength: binary.BigEndian.Uint32(headerBuf[8:12]),
		BodyLength:     binary.BigEndian.Uint32(headerBuf[12:16]),
	}

	if header.MagicNumber != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: %d", header.MagicNumber)
	}
	if header.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// 读取协议数据 + 消息体
	frameBody := make([]byte, header.ProtocolLength+header.BodyLength)
	_, err = io.ReadFull(buf, frameBody)
	if err != nil {
		// 如果是EOF，说明客户端在发送消息体时关闭了连接
		if err == io.EOF {
			return nil, io.EOF
		}
		// 如果是其他错误，比如连接被重置，返回原始错误
		if e, ok := err.(net.Error); ok {
			return nil, e
		}
		// 其他错误，包装错误信息
		return nil, fmt.Errorf("read body error: %v", err)
	}

	// 拼接完整帧
	frame := append(headerBuf, frameBody...)

	return frame, nil
}
