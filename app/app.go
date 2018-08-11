package app

type Component interface {
	Interface() interface{}
	Start() error
	Close()
}