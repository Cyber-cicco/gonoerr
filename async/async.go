package async

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Cyber-cicco/gonoerr/optional"
	"github.com/Cyber-cicco/gonoerr/result"
)

// Future represents an async computation that will eventually produce a Result[T]
type Future[T any] struct {
	done   chan struct{}
	result result.Result[T]
	once   sync.Once
}

// NewFuture creates a new Future from an async function
func NewFuture[T any](fn func() result.Result[T]) *Future[T] {
	f := &Future[T]{
		done: make(chan struct{}),
	}
	
	go func() {
		f.result = fn()
		close(f.done)
	}()
	
	return f
}

// Go creates a Future from a function that returns (T, error)
func Go[T any](fn func() (T, error)) *Future[T] {
	return NewFuture(func() result.Result[T] {
		val, err := fn()
		return result.From(val, err)
	})
}

// Await blocks until the Future completes and returns its Result
func (f *Future[T]) Await() result.Result[T] {
	<-f.done
	return f.result
}

// AwaitWithTimeout waits for the Future with a timeout
func (f *Future[T]) AwaitWithTimeout(timeout time.Duration) result.Result[T] {
	select {
	case <-f.done:
		return f.result
	case <-time.After(timeout):
		return result.Err[T](context.DeadlineExceeded)
	}
}

// Then chains another Future operation, passing the result
func Then[T, V any](f *Future[T], fn func(T) *Future[V]) *Future[V] {
	return NewFuture(func() result.Result[V] {
		res := f.Await()
		if res.IsOk() {
			return fn(res.UnsafeUnwrap()).Await()
		}
		return result.Err[V](res.Error())
	})
}

// Recover handles errors in the Future chain
func (f *Future[T]) Recover(fn func(error) result.Result[T]) *Future[T] {
	return NewFuture(func() result.Result[T] {
		res := f.Await()
		if res.IsErr() {
			return fn(res.Error())
		}
		return res
	})
}

// All waits for all Futures to complete and collects their results
func All[T any](futures ...*Future[T]) *Future[[]T] {
	return NewFuture(func() result.Result[[]T] {
		results := make([]T, len(futures))
		for i, f := range futures {
			res := f.Await()
			if res.IsErr() {
				return result.Err[[]T](res.Error())
			}
			results[i] = res.UnsafeUnwrap()
		}
		return result.Ok(results)
	})
}

// Any returns the first Future to complete successfully
func Any[T any](futures ...*Future[T]) *Future[T] {
	return NewFuture(func() result.Result[T] {
		done := make(chan result.Result[T], len(futures))
		
		for _, f := range futures {
			f := f
			go func() {
				done <- f.Await()
			}()
		}
		
		// Return first successful result
		for i := 0; i < len(futures); i++ {
			res := <-done
			if res.IsOk() {
				return res
			}
		}
		
        fmt.Printf("\"failed\": %v\n", "failed")
		// All failed, return last error
		return result.Err[T](errors.New("all routines failed"))
	})
}

// Race returns the first Future to complete (success or failure)
func Race[T any](futures ...*Future[T]) *Future[T] {
	return NewFuture(func() result.Result[T] {
		done := make(chan result.Result[T], len(futures))
		
		for _, f := range futures {
			f := f
			go func() {
				done <- f.Await()
			}()
		}
		
		return <-done
	})
}

// Channel operations with Result/Optional types

// TryReceive attempts to receive from a channel, returning an Optional
func TryReceive[T any](ch <-chan T) optional.Opt[T] {
	select {
	case val, ok := <-ch:
		if ok {
			return optional.Of(val)
		}
		return optional.Opt[T]{}
	default:
		return optional.Opt[T]{}
	}
}

// ReceiveWithTimeout receives from a channel with timeout
func ReceiveWithTimeout[T any](ch <-chan T, timeout time.Duration) result.Result[T] {
	select {
	case val, ok := <-ch:
		if ok {
			return result.Ok(val)
		}
		return result.Err[T](context.Canceled)
	case <-time.After(timeout):
		return result.Err[T](context.DeadlineExceeded)
	}
}

// Collect gathers results from a channel into a slice
func Collect[T any](ch <-chan result.Result[T]) *Future[[]T] {
	return NewFuture(func() result.Result[[]T] {
		var results []T
		for res := range ch {
			if res.IsErr() {
				return result.Err[[]T](res.Error())
			}
			results = append(results, res.UnsafeUnwrap())
		}
		return result.Ok(results)
	})
}

// Stream creates a channel of Results from multiple async operations
func Stream[T any](fns ...func() result.Result[T]) <-chan result.Result[T] {
	ch := make(chan result.Result[T], len(fns))
	
	go func() {
		defer close(ch)
		var wg sync.WaitGroup
		wg.Add(len(fns))
		
		for _, fn := range fns {
			fn := fn
			go func() {
				defer wg.Done()
				ch <- fn()
			}()
		}
		
		wg.Wait()
	}()
	
	return ch
}

// Parallel executes a function on each element concurrently
func Parallel[T, V any](items []T, fn func(T) result.Result[V]) *Future[[]V] {
	return NewFuture(func() result.Result[[]V] {
		results := make([]result.Result[V], len(items))
		var wg sync.WaitGroup
		wg.Add(len(items))
		
		for i, item := range items {
			i, item := i, item
			go func() {
				defer wg.Done()
				results[i] = fn(item)
			}()
		}
		
		wg.Wait()
		
		// Collect results
		values := make([]V, len(results))
		for i, res := range results {
			if res.IsErr() {
				return result.Err[[]V](res.Error())
			}
			values[i] = res.UnsafeUnwrap()
		}
		
		return result.Ok(values)
	})
}

// Retry attempts an operation multiple times
func Retry[T any](attempts int, delay time.Duration, fn func() result.Result[T]) *Future[T] {
	return NewFuture(func() result.Result[T] {
		var lastErr error
		
		for i := 0; i < attempts; i++ {
			res := fn()
			if res.IsOk() {
				return res
			}
			
			lastErr = res.Error()
			if i < attempts-1 {
				time.Sleep(delay)
			}
		}
		return result.Err[T](lastErr)
	})
}

func Await(fns ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	
	for _, fn := range fns {
		fn := fn
		go func() {
			defer wg.Done()
			fn()
		}()
	}
	
	wg.Wait()
}

// AwaitFutures waits for multiple futures to complete
func AwaitFutures[T any](futures ...*Future[T]) []result.Result[T] {
	results := make([]result.Result[T], len(futures))
	for i, f := range futures {
		results[i] = f.Await()
	}
	return results
}
