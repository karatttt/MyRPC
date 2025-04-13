package codec

import (
	"encoding/json"
	"fmt"
)

// ProtocolData 表示RPC协议数据
type ProtocolData struct {
	ServiceName string `json:"service_name"`
	MethodName  string `json:"method_name"`
	// 可以在这里添加更多字段，如：
	// Version     string `json:"version"`
	// Timeout     int    `json:"timeout"`
	// TraceID     string `json:"trace_id"`
	// 等等
}

// Serialize 将ProtocolData序列化为JSON字节数组
func (p *ProtocolData) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize 从JSON字节数组反序列化为ProtocolData
func DeserializeProtocolData(data []byte) (*ProtocolData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty protocol data")
	}

	var p ProtocolData
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to deserialize protocol data: %v", err)
	}

	return &p, nil
}
