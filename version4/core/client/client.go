package client

import (
	"MyRPC/common"
	"MyRPC/core/async"
	"MyRPC/core/internel"
	"context"
	"fmt"

)

type Client interface {
	// Invoke performs a unary RPC.
	Invoke(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) *common.RPCError
	InvokeAsync(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) (*internel.Future, *common.RPCError)
}

var DefaultClient = NewDefaultClient()

func NewDefaultClient() Client {
	return &client{}
}

type client struct{}

func (c *client) Invoke(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) *common.RPCError {

	// 先根据opt，更新options结构体
	opts := DefaultOptions
	for _, o := range opt {
		o(opts)
	}

	// 发送请求
	if opts.ClientTransport == nil {
		return &common.RPCError{
			Code:    common.ErrCodeClient,
			Message: "clientTransport is nil",
		}
	}
	err := opts.ClientTransport.Send(ctx, reqBody, rspBody, opts.ClientTransportOption)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeClient,
			Message: err.Error(),
		}
	}
	return nil

}
func (c *client) InvokeAsync(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) (*internel.Future, *common.RPCError) {
	future := internel.NewFuture()
	opts := DefaultOptions
	for _, o := range opt {
		o(opts)
	}

	go func() {
		var task *async.Task
		if opts.Timeout > 0 {
			// 有超时时间的情况下，无论是否进行重试，将任务提交给全局管理器
			ctx, msg := internel.GetMessage(ctx)
			task = &async.Task{
				MethodName:  msg.GetMethodName(),
				Request:     reqBody,
				MaxRetries:  opts.RetryTimes,
				Timeout:     opts.Timeout,
				ExecuteFunc: c.makeRetryFunc(ctx, reqBody, rspBody, opts),
				OnComplete: func(err error) {
					// 最终结果回调到原始Future
					if err != nil {
						future.SetResult(nil, &common.RPCError{
							Code:    common.ErrCodeRetryFailed,
							Message: err.Error(),
						})
					} else {
						future.SetResult(rspBody, nil)
					}
				},
			}
			// 提交任务到全局管理器
			task.Status = async.TaskStatusPending
			future.Task = task
			async.GetGlobalTaskManager().AddTask(task)
			// 执行发送逻辑
			err := opts.ClientTransport.Send(ctx, reqBody, rspBody, opts.ClientTransportOption)
			if err == nil {
				future.SetResult(rspBody, nil)
			}
		} else {
			// 无超时时间的情况下，错误的话直接返回
			err := opts.ClientTransport.Send(ctx, reqBody, rspBody, opts.ClientTransportOption)
			if err == nil {
				future.SetResult(rspBody, nil)
			} else {
				future.SetResult(nil, &common.RPCError{
					Code:    common.ErrCodeClient,
					Message: err.Error(),
				})
			}
		}
	}()
	return future, nil
}

func (c *client) makeRetryFunc(
	ctx context.Context,
	reqBody interface{},
	rspBody interface{},
	opts *Options,
) func() error {
	return func() error {
		// 直接使用原始参数发送请求（不递归调用InvokeAsync）
		err := opts.ClientTransport.Send(ctx, reqBody, rspBody, opts.ClientTransportOption)
		if err != nil {
			return fmt.Errorf("RPC调用失败: %v", err)
		}
		return nil
	}
}
