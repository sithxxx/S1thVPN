package tun

type Device interface{
	Read(buf []byte) (int. error)
	Write(pkt []bute) (int, error)
	Name() String
	MTU() int
	Close() error
}