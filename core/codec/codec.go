package codec

import (
	"context"
	"encoding/json"
)

var DefaultCodec = NewCodec()

type codec struct{}

type Codec interface {
	Encode(ctx context.Context, reqBody interface{}) ([]byte, error)
	Decode(ctx context.Context, rspBody []byte) (interface{}, error)
}


func NewCodec() Codec {
	return &codec{}
}

func (c *codec) Encode(ctx context.Context, reqBody interface{}) ([]byte, error) {
	return json.Marshal(reqBody)
}

func (c *codec) Decode(ctx context.Context, rspBody []byte) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal(rspBody, &v)
	return v, err
}


