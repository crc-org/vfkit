package rest

type ServiceScheme int

const (
	Tcp ServiceScheme = iota
	Unix
	None
)
