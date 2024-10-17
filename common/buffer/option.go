package buffer

type Option func(*Buffer)

func WithData(data []byte) Option {
	return func(b *Buffer) {
		b.buf = data
	}
}
