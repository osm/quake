package infostring

type Option func(*InfoString)

func WithKeyValue(key, value string) Option {
	return func(is *InfoString) {
		is.Info = append(is.Info, Info{key, value})
	}
}
