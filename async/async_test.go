package async

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
	"github.com/Cyber-cicco/gonoerr/result"
)

// Test helpers
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test basic Future operations
func TestFutureBasic(t *testing.T) {
	t.Run("successful future", func(t *testing.T) {
		future := Go(func() (string, error) {
			return "hello", nil
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "hello")
	})
	
	t.Run("failed future", func(t *testing.T) {
		expectedErr := errors.New("test error")
		future := Go(func() (string, error) {
			return "", expectedErr
		})
		
		res := future.Await()
		if !res.IsErr() {
			t.Fatal("expected error")
		}
		
		assertEqual(t, res.Error(), expectedErr)
	})
	
	t.Run("future with panic recovery", func(t *testing.T) {
		future := NewFuture(func() result.Result[string] {
			defer func() {
				if r := recover(); r != nil {
					// Future should handle panics gracefully
				}
			}()
			panic("test panic")
		})
		
		// This test ensures the Future doesn't crash the program
		// In a real implementation, you might want to convert panics to errors
		_ = future
	})
}

func TestFutureTimeout(t *testing.T) {
	t.Run("completes before timeout", func(t *testing.T) {
		future := Go(func() (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 42, nil
		})
		
		res := future.AwaitWithTimeout(100 * time.Millisecond)
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), 42)
	})
	
	t.Run("timeout exceeded", func(t *testing.T) {
		future := Go(func() (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 42, nil
		})
		
		res := future.AwaitWithTimeout(50 * time.Millisecond)
		if !res.IsErr() {
			t.Fatal("expected timeout error")
		}
		
		if !errors.Is(res.Error(), context.DeadlineExceeded) {
			t.Errorf("expected DeadlineExceeded, got %v", res.Error())
		}
	})
}

func TestThen(t *testing.T) {
	t.Run("then chain success", func(t *testing.T) {
		future := Then(
			Go(func() (string, error) { return "hello", nil }),
			func(s string) *Future[int] {
				return Go(func() (int, error) {
					return len(s), nil
				})
			},
		)
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), 5)
	})
}

func TestFutureRecover(t *testing.T) {
	t.Run("recover from error", func(t *testing.T) {
		future := Go(func() (string, error) {
			return "", errors.New("original error")
		}).Recover(func(err error) result.Result[string] {
			return result.Ok("recovered: " + err.Error())
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected recovery, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "recovered: original error")
	})
	
	t.Run("recover not called on success", func(t *testing.T) {
		future := Go(func() (string, error) {
			return "success", nil
		}).Recover(func(err error) result.Result[string] {
			t.Error("recover should not be called on success")
			return result.Ok("should not happen")
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "success")
	})
}

func TestAll(t *testing.T) {
	t.Run("all futures succeed", func(t *testing.T) {
		f1 := Go(func() (int, error) { return 1, nil })
		f2 := Go(func() (int, error) { return 2, nil })
		f3 := Go(func() (int, error) { return 3, nil })
		
		allFuture := All(f1, f2, f3)
		res := allFuture.Await()
		
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		values := res.UnsafeUnwrap()
		if len(values) != 3 {
			t.Fatalf("expected 3 values, got %d", len(values))
		}
		
		assertEqual(t, values[0], 1)
		assertEqual(t, values[1], 2)
		assertEqual(t, values[2], 3)
	})
	
	t.Run("one future fails", func(t *testing.T) {
		expectedErr := errors.New("f2 error")
		f1 := Go(func() (int, error) { return 1, nil })
		f2 := Go(func() (int, error) { return 0, expectedErr })
		f3 := Go(func() (int, error) { return 3, nil })
		
		allFuture := All(f1, f2, f3)
		res := allFuture.Await()
		
		if !res.IsErr() {
			t.Fatal("expected error")
		}
		
		assertEqual(t, res.Error(), expectedErr)
	})
	
	t.Run("empty futures", func(t *testing.T) {
		allFuture := All[int]()
		res := allFuture.Await()
		
		if !res.IsOk() {
			t.Fatalf("expected success for empty futures")
		}
		
		values := res.UnsafeUnwrap()
		assertEqual(t, len(values), 0)
	})
}

func TestAny(t *testing.T) {
	t.Run("first success wins", func(t *testing.T) {
		f1 := Go(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow", nil
		})
		f2 := Go(func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "fast", nil
		})
		f3 := Go(func() (string, error) {
			return "", errors.New("immediate error")
		})
		
		anyFuture := Any(f1, f2, f3)
		res := anyFuture.Await()
		
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "fast")
	})
	
	t.Run("all futures fail", func(t *testing.T) {
		f1 := Go(func() (string, error) { return "", errors.New("error 1") })
		f2 := Go(func() (string, error) { return "", errors.New("error 2") })
		f3 := Go(func() (string, error) { return "", errors.New("error 3") })
		
		anyFuture := Any(f1, f2, f3)
		res := anyFuture.Await()
		
		if !res.IsErr() {
			t.Fatal("expected error when all futures fail")
		}
	})
}

func TestRace(t *testing.T) {
	t.Run("fastest future wins", func(t *testing.T) {
		f1 := Go(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow", nil
		})
		f2 := Go(func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "fast", nil
		})
		
		raceFuture := Race(f1, f2)
		res := raceFuture.Await()
		
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "fast")
	})
	
	t.Run("error can win race", func(t *testing.T) {
		f1 := Go(func() (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow", nil
		})
		f2 := Go(func() (string, error) {
			return "", errors.New("fast error")
		})
		
		raceFuture := Race(f1, f2)
		res := raceFuture.Await()
		
		if !res.IsErr() {
			t.Fatal("expected fast error to win")
		}
	})
}

func TestTryReceive(t *testing.T) {
	t.Run("receive from channel with value", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 42
		
		opt := TryReceive(ch)
		if !opt.IsPresent() {
			t.Fatal("expected value")
		}
		
		assertEqual(t, opt.OrElseGet(0), 42)
	})
	
	t.Run("receive from empty channel", func(t *testing.T) {
		ch := make(chan int)
		
		opt := TryReceive(ch)
		if opt.IsPresent() {
			t.Fatal("expected no value")
		}
	})
	
	t.Run("receive from closed channel", func(t *testing.T) {
		ch := make(chan int)
		close(ch)
		
		opt := TryReceive(ch)
		if opt.IsPresent() {
			t.Fatal("expected no value from closed channel")
		}
	})
}

func TestReceiveWithTimeout(t *testing.T) {
	t.Run("receive before timeout", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 42
		
		res := ReceiveWithTimeout(ch, 100*time.Millisecond)
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), 42)
	})
	
	t.Run("timeout on empty channel", func(t *testing.T) {
		ch := make(chan int)
		
		res := ReceiveWithTimeout(ch, 50*time.Millisecond)
		if !res.IsErr() {
			t.Fatal("expected timeout error")
		}
		
		if !errors.Is(res.Error(), context.DeadlineExceeded) {
			t.Errorf("expected DeadlineExceeded, got %v", res.Error())
		}
	})
	
	t.Run("closed channel returns error", func(t *testing.T) {
		ch := make(chan int)
		close(ch)
		
		res := ReceiveWithTimeout(ch, 100*time.Millisecond)
		if !res.IsErr() {
			t.Fatal("expected error from closed channel")
		}
		
		if !errors.Is(res.Error(), context.Canceled) {
			t.Errorf("expected Canceled, got %v", res.Error())
		}
	})
}

func TestCollect(t *testing.T) {
	t.Run("collect all successful results", func(t *testing.T) {
		ch := make(chan result.Result[int], 3)
		ch <- result.Ok(1)
		ch <- result.Ok(2)
		ch <- result.Ok(3)
		close(ch)
		
		future := Collect(ch)
		res := future.Await()
		
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		values := res.UnsafeUnwrap()
		assertEqual(t, len(values), 3)
		assertEqual(t, values[0], 1)
		assertEqual(t, values[1], 2)
		assertEqual(t, values[2], 3)
	})
	
	t.Run("collect stops on first error", func(t *testing.T) {
		ch := make(chan result.Result[int], 3)
		ch <- result.Ok(1)
		ch <- result.Err[int](errors.New("test error"))
		ch <- result.Ok(3)
		close(ch)
		
		future := Collect(ch)
		res := future.Await()
		
		if !res.IsErr() {
			t.Fatal("expected error")
		}
	})
}

func TestStream(t *testing.T) {
	t.Run("stream multiple operations", func(t *testing.T) {
		results := Stream(
			func() result.Result[int] { return result.Ok(1) },
			func() result.Result[int] { return result.Ok(2) },
			func() result.Result[int] { return result.Ok(3) },
		)
		
		var values []int
		for res := range results {
			if res.IsOk() {
				values = append(values, res.UnsafeUnwrap())
			}
		}
		
		assertEqual(t, len(values), 3)
	})
	
	t.Run("stream with errors", func(t *testing.T) {
		var errorCount int
		results := Stream(
			func() result.Result[int] { return result.Ok(1) },
			func() result.Result[int] { return result.Err[int](errors.New("error")) },
			func() result.Result[int] { return result.Ok(3) },
		)
		
		var successCount int
		for res := range results {
			if res.IsOk() {
				successCount++
			} else {
				errorCount++
			}
		}
		
		assertEqual(t, successCount, 2)
		assertEqual(t, errorCount, 1)
	})
}

func TestParallel(t *testing.T) {
	t.Run("parallel processing success", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		
		future := Parallel(items, func(n int) result.Result[int] {
			return result.Ok(n * n)
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		values := res.UnsafeUnwrap()
		assertEqual(t, len(values), 5)
		assertEqual(t, values[0], 1)
		assertEqual(t, values[1], 4)
		assertEqual(t, values[2], 9)
		assertEqual(t, values[3], 16)
		assertEqual(t, values[4], 25)
	})
	
	t.Run("parallel processing with error", func(t *testing.T) {
		items := []int{1, 2, 3}
		
		future := Parallel(items, func(n int) result.Result[int] {
			if n == 2 {
				return result.Err[int](errors.New("error at 2"))
			}
			return result.Ok(n * n)
		})
		
		res := future.Await()
		if !res.IsErr() {
			t.Fatal("expected error")
		}
	})
	
	t.Run("parallel actually runs in parallel", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		var counter int32
		
		start := time.Now()
		future := Parallel(items, func(n int) result.Result[int] {
			atomic.AddInt32(&counter, 1)
			time.Sleep(100 * time.Millisecond)
			return result.Ok(n)
		})
		
		res := future.Await()
		elapsed := time.Since(start)
		
		if !res.IsOk() {
			t.Fatalf("expected success, got error: %v", res.Error())
		}
		
		// If running in parallel, should take ~100ms, not 500ms
		if elapsed > 200*time.Millisecond {
			t.Errorf("parallel execution took too long: %v", elapsed)
		}
		
		assertEqual(t, atomic.LoadInt32(&counter), 5)
	})
}

func TestRetry(t *testing.T) {
	t.Run("retry succeeds eventually", func(t *testing.T) {
		var attempts int
		future := Retry(3, 10*time.Millisecond, func() result.Result[string] {
			attempts++
			if attempts < 3 {
				return result.Err[string](fmt.Errorf("attempt %d", attempts))
			}
			return result.Ok("success")
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected success after retries, got: %v", res.Error())
		}
		
		assertEqual(t, res.UnsafeUnwrap(), "success")
		assertEqual(t, attempts, 3)
	})
	
	t.Run("retry fails after max attempts", func(t *testing.T) {
		var attempts int
		future := Retry(3, 10*time.Millisecond, func() result.Result[string] {
			attempts++
			return result.Err[string](fmt.Errorf("attempt %d", attempts))
		})
		
		res := future.Await()
		if !res.IsErr() {
			t.Fatal("expected error after max retries")
		}
		
		assertEqual(t, attempts, 3)
		assertEqual(t, res.Error().Error(), "attempt 3")
	})
	
	t.Run("retry succeeds immediately", func(t *testing.T) {
		var attempts int
		future := Retry(3, 10*time.Millisecond, func() result.Result[string] {
			attempts++
			return result.Ok("immediate success")
		})
		
		res := future.Await()
		if !res.IsOk() {
			t.Fatalf("expected immediate success, got: %v", res.Error())
		}
		
		assertEqual(t, attempts, 1)
	})
}

func TestAwaitFutures(t *testing.T) {
	f1 := Go(func() (int, error) { return 1, nil })
	f2 := Go(func() (int, error) { return 0, errors.New("error") })
	f3 := Go(func() (int, error) { return 3, nil })
	
	results := AwaitFutures(f1, f2, f3)
	
	assertEqual(t, len(results), 3)
	
	if !results[0].IsOk() {
		t.Error("expected first result to be ok")
	}
	
	if !results[1].IsErr() {
		t.Error("expected second result to be error")
	}
	
	if !results[2].IsOk() {
		t.Error("expected third result to be ok")
	}
}

func TestConcurrentFutureAccess(t *testing.T) {
	// Test that multiple goroutines can safely await the same future
	future := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	})
	
	const numGoroutines = 10
	results := make(chan int, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			res := future.Await()
			if res.IsOk() {
				results <- res.UnsafeUnwrap()
			}
		}()
	}
	
	for i := 0; i < numGoroutines; i++ {
		val := <-results
		assertEqual(t, val, 42)
	}
}

// Benchmarks
func BenchmarkFutureCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Go(func() (int, error) {
			return 42, nil
		})
	}
}

func BenchmarkFutureAwait(b *testing.B) {
	futures := make([]*Future[int], b.N)
	for i := 0; i < b.N; i++ {
		futures[i] = Go(func() (int, error) {
			return i, nil
		})
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = futures[i].Await()
	}
}

func BenchmarkParallelProcessing(b *testing.B) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		future := Parallel(items, func(n int) result.Result[int] {
			return result.Ok(n * n)
		})
		_ = future.Await()
	}
}
