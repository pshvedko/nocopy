package api

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	Chains []uuid.UUID `json:"chains,omitempty"`
	Blocks []uuid.UUID `json:"blocks,omitempty"`
	Hashes [][]byte    `json:"hashes,omitempty"`
	Sizes  []int64     `json:"sizes,omitempty"`
}

type FileReply struct {
	Time time.Time `json:"time"`
}

type Echo struct {
	Serial int           `json:"serial"`
	Delay  time.Duration `json:"delay"`
}

type Head struct {
	Name string `json:"name,omitempty"`
}

type HeadReply struct {
	Name   string      `json:"name,omitempty"`
	Time   time.Time   `json:"time"`
	Size   int64       `json:"size,omitempty"`
	Blocks []uuid.UUID `json:"blocks,omitempty"`
	Sizes  []int64     `json:"sizes,omitempty"`
}

func (x *HeadReply) GetLength() int64 {
	var size int64
	for i := range x.Sizes {
		size += x.Sizes[i]
	}
	return size
}
