package service

import (
	"context"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker/message"
)

func (s *Block) Head(w http.ResponseWriter, r *http.Request) {
	s.Add(1)
	defer s.Done()
	_, reply, err := s.Broker.Request(r.Context(), "proxy", "head", &api.Head{Name: path.Clean(r.URL.Path)})
	if err == nil {
		var head api.HeadReply
		err = reply.Unmarshal(&head)
		if err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(head.GetLength(), 10))
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	slog.Error("get", "err", err)
}

func (s *Proxy) HeadQuery(ctx context.Context, m message.Message) (any, error) {
	s.Add(1)
	defer s.Done()
	_, err := s.Broker.Send(ctx, message.Forward{To: "chain", Message: m})
	return nil, err
}

func (s *Proxy) HeadReply(ctx context.Context, m message.Message) {
	s.Add(1)
	defer s.Done()
	_, _ = s.Broker.Send(ctx, message.Backward{Message: m})
}

func (s *Chain) HeadQuery(ctx context.Context, m message.Message) (any, error) {
	s.Add(1)
	defer s.Done()
	var head api.Head
	err := m.Unmarshal(&head)
	if err != nil {
		return nil, err
	}
	slog.Warn("head", "name", head.Name)
	_, _, _, blocks, size, err := s.Repository.Get(ctx, head.Name)
	if err != nil {
		return nil, err
	}
	var block [][]byte
	for i := range blocks {
		block = append(block, blocks[i][:])
	}
	return &api.HeadReply{
		Block: block,
		Size:  size,
	}, nil
}
