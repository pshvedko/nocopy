package service

import (
	"context"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"

	"github.com/pshvedko/nocopy/broker/message"
)

var ErrDigestMismatch = errors.New("digest do not match")
var ErrAlgorithmMismatch = errors.New("algorithm do not match")

type ContextKey struct {
	name string
}

func (k *ContextKey) String() string {
	return k.name
}

var AuthorizeKey = &ContextKey{"Authorize"}

type Authorize struct {
	User string `json:"user,omitempty"`
}

func (a Authorize) String() string { return fmt.Sprintf("%#v", a) }

func (a Authorize) Name() string {
	return "!authorize"
}

func (a Authorize) Writer(w message.Writer) message.Writer {
	return w
}

func (a Authorize) Decode(ctx context.Context, headers map[string][]string, bytes []byte, begin int, size int) (context.Context, error) {
	err := json.Unmarshal(bytes[begin:begin+size], &a)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, AuthorizeKey, &a), nil
}

func (a Authorize) Encode(ctx context.Context) ([]byte, error) {
	x, ok := ctx.Value(AuthorizeKey).(*Authorize)
	if ok {
		return json.Marshal(x)
	}
	return json.Marshal(a)
}

type Digest struct {
	message.Writer
	hash.Hash
	A string
}

func (d *Digest) Write(p []byte) (int, error) {
	n, err := d.Writer.Write(p)
	if err != nil {
		return n, err
	}
	return d.Hash.Write(p[:n])
}

func (d *Digest) Bytes() []byte {
	return d.Writer.Bytes()
}

func (d *Digest) Len() int {
	return d.Writer.Len()
}

func (d *Digest) Headers() map[string][]string {
	return Append(map[string][]string{"Digest": {hex.EncodeToString(d.Hash.Sum([]byte{}))}}, d.Writer.Headers())
}

type Signature struct {
	Algorithm crypto.Hash `json:"algorithm"`
}

func (s Signature) Name() string {
	return "!signature"
}

func (s Signature) Writer(w message.Writer) message.Writer {
	return &Digest{
		A:      s.Algorithm.String(),
		Hash:   s.Algorithm.New(),
		Writer: w,
	}
}

func (s Signature) Decode(ctx context.Context, headers map[string][]string, bytes []byte, begin int, size int) (context.Context, error) {
	a := s.Algorithm
	err := json.Unmarshal(bytes[begin:begin+size], &s)
	if err != nil {
		return nil, err
	}
	if a != s.Algorithm {
		return nil, ErrAlgorithmMismatch
	}
	h := s.Algorithm.New()
	_, err = h.Write(bytes)
	if err != nil {
		return nil, err
	}
	x, ok := headers["Digest"]
	if !ok || len(x) == 0 || x[0] != hex.EncodeToString(h.Sum([]byte{})) {
		return nil, ErrDigestMismatch
	}
	return ctx, nil
}

func (s Signature) Encode(context.Context) ([]byte, error) {
	return json.Marshal(s)
}

func Append(a map[string][]string, b map[string][]string) map[string][]string {
	for k, v := range b {
		a[k] = append(a[k], v...)
	}
	return a
}
