package message

type Query interface {
	Unmarshal(any) error
}

type Reply interface {
	Marshal() ([]byte, error)
}
