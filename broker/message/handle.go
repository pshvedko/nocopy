package message

import "context"

type Handler func(context.Context, Message) (any, error)

type Catcher func(context.Context, Message)
