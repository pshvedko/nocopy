package message

import "context"

type Handler func(context.Context, Query) (any, error)

type Catcher func(context.Context, Query)
