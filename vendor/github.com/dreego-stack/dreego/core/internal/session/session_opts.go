package session

func opt[T any](opts *Options, fn func(*Options) T) T {
	if opts == nil {
		var zero T
		return zero
	}
	return fn(opts)
}
