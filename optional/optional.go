package optional

import "github.com/Cyber-cicco/gonoerr/result"

type Opt[T any] struct {
	content *T
}

func Of[T any](val T) Opt[T] {
	return Opt[T]{
		content: &val,
	}
}

func OfNullable[T any](ptr *T) Opt[T] {
	if ptr != nil {
		return Opt[T]{
			content: ptr,
		}
	}
	return Opt[T]{}
}

func Map[T any, V any](opt Opt[T], fn func(*T) V) Opt[V] {
	if opt.content != nil {
		res := fn(opt.content)
		return Of(res)
	}
	return Opt[V]{}
}

func FlatMap[T any, V any](opt Opt[T], fn func(*T) Opt[V]) Opt[V] {
	if opt.content != nil {
		return fn(opt.content)
	}
	return Opt[V]{}
}

func (o Opt[T]) IfPresentOrElse(present func(*T), absent func()) Opt[T] {
	if o.content != nil {
		present(o.content)
	} else {
		absent()
	}
	return o
}

func (o Opt[T]) IfPresent(present func(*T)) Opt[T] {
	if o.content != nil {
		present(o.content)
	}
	return o
}

func (o Opt[T]) OrElseDo(fn func()) Opt[T] {
	if o.content == nil {
		fn()
	}
	return o
}

func (o Opt[T]) OrElseGet(val T) T {
	if o.content == nil {
		return val
	}
	return *o.content
}

func (o Opt[T]) IsPresent() bool {
	return o.content != nil
}

func (o Opt[T]) Filter(fn func(*T) bool) Opt[T] {
	if o.content != nil {
		ok := fn(o.content)
		if ok {
			return o
		}
	}
	return Opt[T]{}
}

func (o Opt[T]) AsResult(err error) result.Result[*T] {
	if o.content != nil {
		return result.From(o.content, nil)
	}
	return result.From[*T](nil, err)
}
