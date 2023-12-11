package block

import "github.com/google/uuid"

type Block struct {
	ID   uuid.UUID
	Hash []byte
	Size int64
}
