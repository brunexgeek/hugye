package domain

type Resolver interface {
	Send(buf []byte, id uint16) (uint16, error)
	Receive(timeout int) ([]byte, error)
	NextId() uint16
}

type Cache interface {
	Get(name string, typ uint16) []byte
	Set(name string, typ uint16, buf []byte)
}
