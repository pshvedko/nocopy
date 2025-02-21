package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Hash []byte

func (h Hash) String() string {
	var b strings.Builder
	for _, c := range h {
		_, _ = fmt.Fprintf(&b, "%02x", c)
	}
	return b.String()
}

type File struct {
	Chains []uuid.UUID `json:"chains,omitempty"`
	Blocks []uuid.UUID `json:"blocks,omitempty"`
	Hashes []Hash      `json:"hashes,omitempty"`
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
