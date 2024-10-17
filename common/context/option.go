package context

type Option func(*Context)

func WithIsDem(isDem bool) Option {
	return func(ctx *Context) {
		ctx.isDem = isDem
	}
}

func WithIsQWD(isQWD bool) Option {
	return func(ctx *Context) {
		ctx.isQWD = isQWD
	}
}

func WithIsMVD(isMVD bool) Option {
	return func(ctx *Context) {
		ctx.isMVD = isMVD
	}
}

func WithProtocolVersion(protocolVersion uint32) Option {
	return func(ctx *Context) {
		ctx.protocolVersion = protocolVersion
	}
}
