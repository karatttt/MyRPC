package codec

import (
	"fmt"
	"google.golang.org/protobuf/proto"
)


// Marshal 将结构体序列化为 protobuf 字节数组
func Marshal(reqBody interface{}) ([]byte, error) {
	msg, ok := reqBody.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("Marshal: reqBody does not implement proto.Message")
	}
	return proto.Marshal(msg)
}

// Unmarshal 将 protobuf 字节数组反序列化为结构体
func Unmarshal(rspDataBuf []byte, rspBody interface{}) error {
	msg, ok := rspBody.(proto.Message)
	if !ok {
		return fmt.Errorf("Unmarshal: rspBody does not implement proto.Message")
	}
	return proto.Unmarshal(rspDataBuf, msg)
}
