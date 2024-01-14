package api

import (
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
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

func (x *HeadReply) GetBlockUUID() uuid.UUIDs {
	var blocks uuid.UUIDs
	for _, bytes := range x.Block {
		id, err := uuid.FromBytes(bytes)
		if err != nil {
			continue
		}
		blocks = append(blocks, id)
	}
	return blocks
}

func (x *HeadReply) GetLength() int64 {
	var size int64
	for i := range x.Size {
		size += x.Size[i]
	}
	return size
}
