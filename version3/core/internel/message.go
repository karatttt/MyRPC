package internel	

import "context"

type Message interface{
	GetServiceName() string
	GetMethodName() string
	GetMuxOpen() bool
	WithServiceName(serviceName string) 
	WithMethodName(methodName string) 
	WithMuxOpen(muxOpen bool)
	GetSequenceID() uint32
	WithSequenceID(sequenceID uint32)

} 

// 为什么要有msg这个结构，目前的设计是msg是跟service相关数据，而opts是每一次调用时，client传入的参数，相对来说后者更个性，前者更通用
type msg struct {
	ServiceName string
	MethodName  string
	muxOpen     bool
	SequenceID  uint32
}

// 为了避免不同的包中使用相同的context key导致冲突，使用自定义的context key
type ContextKey string

const (
	ContextMsgKey ContextKey = ContextKey("MyRPC")
)

// Message returns the message of context.
func GetMessage(ctx context.Context) (context.Context, Message) {
	val := ctx.Value(ContextMsgKey)
	if m, ok := val.(*msg); ok {
		return ctx, m
	}
	newMsg := &msg{}
	newCtx := context.WithValue(ctx, ContextMsgKey, newMsg)
	return newCtx, newMsg
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

func (m *msg) GetMuxOpen() bool {
	return m.muxOpen
}
func (m *msg) WithMuxOpen(muxOpen bool) {
	m.muxOpen = muxOpen
}
func (m *msg) GetSequenceID() uint32 {
	return m.SequenceID
}	
func (m *msg) WithSequenceID(sequenceID uint32) {
	m.SequenceID = sequenceID
}