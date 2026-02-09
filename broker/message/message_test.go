package message_test

import (
	"context"
	"crypto"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/service"
)

type Mediator struct {
	Middlewares []message.Middleware
}

func (m Mediator) Middleware(string) []message.Middleware {
	return m.Middlewares
}

func TestEncode(t *testing.T) {
	type args struct {
		ctx      context.Context
		m        message.Message
		mediator message.Mediator
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]string
		want1   []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				ctx:      context.WithValue(context.TODO(), service.AuthorizeKey, &service.Authorize{User: "admin"}),
				m:        message.NewWithID(uuid.UUID{}).WithMethod("try").WithTo("me").WithFrom("left").WithBody(api.Echo{Serial: 1, Delay: time.Second}).Forward("right"),
				mediator: Mediator{Middlewares: []message.Middleware{service.Authorize{}, service.Signature{Algorithm: crypto.SHA256}}},
			},
			want:    map[string][]string{"Digest": {`1c713e033f30e90d5ad615cf9637e4c4d7dd5d28dfdcfa14a146717819cee7bc`}},
			want1:   []byte((`[{"id":"00000000-0000-0000-0000-000000000000","from":"me","return":["left"],"to":"right","method":"try","use":["!authorize","!signature"]},{"user":"admin"},{"algorithm":5},{"serial":1,"delay":1000000000}]`)),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := message.Encode(tt.args.ctx, tt.args.m, tt.args.mediator)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Encode() got1 = %s, want %s", got1, tt.want1)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	type args struct {
		ctx      context.Context
		headers  map[string][]string
		bytes    []byte
		mediator message.Mediator
	}
	tests := []struct {
		name    string
		args    args
		want    context.Context
		want1   message.Message
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				ctx:      context.TODO(),
				headers:  map[string][]string{"Digest": {`1c713e033f30e90d5ad615cf9637e4c4d7dd5d28dfdcfa14a146717819cee7bc`}},
				bytes:    []byte((`[{"id":"00000000-0000-0000-0000-000000000000","from":"me","return":["left"],"to":"right","method":"try","use":["!authorize","!signature"]},{"user":"admin"},{"algorithm":5},{"serial":1,"delay":1000000000}]`)),
				mediator: Mediator{Middlewares: []message.Middleware{service.Authorize{User: "admin"}, service.Signature{Algorithm: crypto.SHA256}}},
			},
			want: context.WithValue(context.TODO(), service.AuthorizeKey, &service.Authorize{User: "admin"}),
			want1: message.Raw{
				Err: message.Error{},
				Envelope: message.Envelope{
					ID:     uuid.UUID{},
					From:   "me",
					Return: []string{"left"},
					To:     "right",
					Type:   0,
					Method: "try",
					Use:    []string{"!authorize", "!signature"},
				},
				Body: []byte(`{"serial":1,"delay":1000000000}`),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := message.Decode(tt.args.ctx, tt.args.headers, tt.args.bytes, tt.args.mediator)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Decode() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
