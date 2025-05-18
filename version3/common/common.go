package common

import "fmt"

// RPCError 定义RPC错误结构
type RPCError struct {
	Code    ErrorCode
	Message string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error: [%s] %s", e.Code, e.Message)
}

// ErrorCode 定义错误码类型
type ErrorCode string

const (
	ErrCodeTimeout         ErrorCode = "TIMEOUT"
	ErrCodeConnection      ErrorCode = "CONNECTION_ERROR"
	ErrCodeNetwork         ErrorCode = "NETWORK_ERROR"
	ErrCodeSerialization   ErrorCode = "SERIALIZATION_ERROR"
	ErrCodeDeserialization ErrorCode = "DESERIALIZATION_ERROR"
	ErrCodeEncoding        ErrorCode = "ENCODING_ERROR"
	ErrCodeDecoding        ErrorCode = "DECODING_ERROR"
	ErrCodeServer          ErrorCode = "SERVER_ERROR"
	ErrCodeClient          ErrorCode = "CLIENT_ERROR"
	ErrCodeRetryFailed     ErrorCode = "RETRY_FAILED"
)

// 成功
var ErrCodeSuccess = &RPCError{
	Code:    "SUCCESS",
	Message: "success",
}
