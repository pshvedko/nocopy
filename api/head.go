package api

import (
	"github.com/golang/protobuf/proto"
)

func (x *Head) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *Head) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, x)
}

func (x *HeadReply) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *HeadReply) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, x)
}
