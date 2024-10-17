package packet

type Packet interface {
	Bytes() []byte
}
