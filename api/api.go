package api

import (
	"github.com/google/uuid"
)

type File struct {
	Chains []uuid.UUID `json:"chains,omitempty"`
	Blocks []uuid.UUID `json:"blocks,omitempty"`
	Hashes [][]byte    `json:"hashes,omitempty"`
	Sizes  []int64     `json:"sizes,omitempty"`
}
