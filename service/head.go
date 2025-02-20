package service

import (
	"context"
	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker/message"
	"log/slog"
	"net/http"
	"path"
	"strconv"
)

func (s *Block) Head(w http.ResponseWriter, r *http.Request) {
	reply, err := s.Broker.Request(r.Context(), "proxy", "head", message.NewBody(api.Head{Name: path.Clean(r.URL.Path)}))
	if err == nil {
		var head api.HeadReply
		err = reply.Decode(&head)
		if err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(head.GetLength(), 10))
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	slog.Error("head", "err", err)
}

func (s *Proxy) HeadQuery(ctx context.Context, m message.Message) (message.Body, error) {
	_, err := s.Broker.Forward(ctx, "chain", m)
	return nil, err
}

func (s *Proxy) HeadReply(ctx context.Context, m message.Message) bool {
	_, _ = s.Broker.Backward(ctx, m)
	return true
}

func (s *Chain) HeadQuery(ctx context.Context, m message.Message) (message.Body, error) {
	var head api.Head
	err := m.Decode(&head)
	if err != nil {
		return nil, err
	}
	slog.Warn("head", "name", head.Name)
	name, date, length, blocks, sizes, err := s.Repository.Get(ctx, head.Name)
	if err != nil {
		return nil, err
	}
	return message.NewBody(api.HeadReply{
		Name:   name,
		Time:   date,
		Size:   length,
		Blocks: blocks,
		Sizes:  sizes,
	}), nil
}
