package internel	

import "context"

type Message interface{
	GetServiceName() string
	GetMethodName() string
	WithServiceName(serviceName string) 
	WithMethodName(methodName string) 
} 

// 为什么要有msg这个结构，目前的设计是msg是跟service相关数据，而opts是每一次调用时，client传入的参数，相对来说后者更个性，前者更通用
type msg struct {
	ServiceName string
	MethodName  string
}

// 为了避免不同的包中使用相同的context key导致冲突，使用自定义的context key
type ContextKey string

const (
	ContextMsgKey ContextKey = ContextKey("MyRPC")
)

// Message returns the message of context.
func GetMessage(ctx context.Context) Message {
	val := ctx.Value(ContextMsgKey)
	m, ok := val.(*msg)
	if !ok {
		return &msg{}
	}
	return m
}

func NewMsg() *msg {
	return &msg{}
}

func (m *msg) GetServiceName() string {
	return m.ServiceName
}

func (m *msg) GetMethodName() string {
	return m.MethodName
}

func (m *msg) WithServiceName(serviceName string) {
	m.ServiceName = serviceName
}

func (m *msg) WithMethodName(methodName string) {
	m.MethodName = methodName
}
