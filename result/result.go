package result

type Result[T any] struct {
	ok  T
	err error
}

type Ok[T any] func(ok T)
type Err func(err error)

func From[T any](ok T, err error) Result[T] {
	return Result[T]{
		ok:  ok,
		err: err,
	}
}

func Map[T any, V any](res Result[T], fn func(T) V) Result[V] {
	if res.IsOk() {
		return Result[V]{ ok: fn(res.ok), err: nil }
	}
	return Result[V]{
		err: res.err,
	}
}

func FlatMap[T any, V any](r Result[T], fn func(T) Result[V]) Result[V] {
    if r.err != nil {
        return Result[V]{err: r.err}
    }
    return fn(r.ok) 
}

func (r Result[T]) UnsafeUnwrap() T {
    if r.err != nil {
        panic(r.err)
    }
    return r.ok
}

func (r Result[T]) AndThen(fn Ok[T]) Result[T] {
	if r.err == nil {
		fn(r.ok)
	}
	return r
}

func (r Result[T]) IfErr(fn func(err error)) Result[T] {
	if r.err != nil {
		fn(r.err)
	}
	return r
}

func (r Result[T]) IsErr() bool {
	return r.err != nil
}

func (r Result[T]) IsOk() bool {
	return r.err == nil
}

func (r Result[T]) Match(ok Ok[T], err Err) Result[T] {
	if r.err != nil {
		err(r.err)
	} else {
		ok(r.ok)
	}
	return r
}

func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err != nil {
		return Result[T]{ok: r.ok, err: fn(r.err)}
	}
	return r
}
