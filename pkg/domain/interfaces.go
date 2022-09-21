package domain

type Cache interface {
	Get(name string, typ uint16) []byte
	Set(name string, typ uint16, buf []byte)
}

type Tree interface {
	Match(host string) bool
}
