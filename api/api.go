package api

//go:generate protoc -I . --go_out=. head.proto

import (
	"github.com/google/uuid"
	"time"
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
	Serial int `json:"serial"`
}
