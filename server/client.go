package server

type Client interface {
	GetAddr() string
	GetName() string
}
