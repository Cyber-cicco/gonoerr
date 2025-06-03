package main

import (
	"errors"
	"fmt"
	"time"
	
	"github.com/Cyber-cicco/gonoerr/async"
	"github.com/Cyber-cicco/gonoerr/result"
)

func main() {
	// Example 1: Basic Future usage
	fmt.Println("=== Basic Future ===")
	future := async.Go(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello, Future!", nil
	})
	
	res := future.Await()
	if res.IsOk() {
		fmt.Println(res.UnsafeUnwrap())
	}
	
	
	
	// Example 2: Error handling with Recover
	fmt.Println("\n=== Error Recovery ===")
	errorFuture := async.Go(func() (string, error) {
		return "", errors.New("something went wrong")
	}).Recover(func(err error) result.Result[string] {
		return result.Ok(fmt.Sprintf("Recovered from: %v", err))
	})
	
	recoveredRes := errorFuture.Await()
	fmt.Println(recoveredRes.UnsafeUnwrap())
	
	// Example 3: Parallel processing
	fmt.Println("\n=== Parallel Processing ===")
	numbers := []int{1, 2, 3, 4, 5}
	parallelFuture := async.Parallel(numbers, func(n int) result.Result[int] {
		// Simulate some work
		time.Sleep(time.Duration(n*10) * time.Millisecond)
		return result.Ok(n * n)
	})
	
	parallelRes := parallelFuture.Await()
	if parallelRes.IsOk() {
		fmt.Printf("Squared numbers: %v\n", parallelRes.UnsafeUnwrap())
	}
	
	// Example 4: Waiting for multiple futures
	fmt.Println("\n=== Multiple Futures ===")
	f1 := async.Go(func() (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "First", nil
	})
	
	f2 := async.Go(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Second", nil
	})
	
	f3 := async.Go(func() (string, error) {
		time.Sleep(75 * time.Millisecond)
		return "Third", nil
	})
	
	allFuture := async.All(f1, f2, f3)
	allRes := allFuture.Await()
	if allRes.IsOk() {
		fmt.Printf("All results: %v\n", allRes.UnsafeUnwrap())
	}
	
	// Example 5: Race condition - first to complete wins
	fmt.Println("\n=== Race Condition ===")
	raceFuture := async.Race(
		async.Go(func() (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "Slow", nil
		}),
		async.Go(func() (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "Fast", nil
		}),
	)
	
	raceRes := raceFuture.Await()
	fmt.Printf("Winner: %v\n", raceRes.UnsafeUnwrap())
	
	// Example 6: Retry mechanism
	fmt.Println("\n=== Retry Mechanism ===")
	attemptCount := 0
	retryFuture := async.Retry(3, 100*time.Millisecond, func() result.Result[string] {
		attemptCount++
		if attemptCount < 3 {
			return result.Err[string](fmt.Errorf("attempt %d failed", attemptCount))
		}
		return result.Ok("Success on third try!")
	})
	
	retryRes := retryFuture.Await()
	fmt.Println(retryRes.UnsafeUnwrap())
	
	// Example 7: Channel operations
	fmt.Println("\n=== Channel Operations ===")
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	
	// Using TryReceive
	for {
		opt := async.TryReceive(ch)
		if opt.IsPresent() {
			fmt.Printf("Received: %v\n", opt.OrElseGet(0))
		} else {
			break
		}
	}
	
	// Example 8: Stream processing
	fmt.Println("\n=== Stream Processing ===")
	streamResults := async.Stream(
		func() result.Result[int] { return result.Ok(1) },
		func() result.Result[int] { return result.Ok(2) },
		func() result.Result[int] { return result.Ok(3) },
		func() result.Result[int] { return result.Err[int](errors.New("skip")) },
		func() result.Result[int] { return result.Ok(5) },
	)
	
	collectFuture := async.Collect(streamResults)
	collectRes := collectFuture.Await()
	if collectRes.IsErr() {
		fmt.Printf("Stream error: %v\n", collectRes.Error())
	}
	
	// Example 9: Pipeline processing
	fmt.Println("\n=== Pipeline Processing ===")
	input := make(chan int)
	go func() {
		for i := 1; i <= 5; i++ {
			input <- i
		}
		close(input)
	}()
	
}

// Example of using Future with existing Result/Optional types
func fetchUserData(id int) *async.Future[User] {
	return async.Go(func() (User, error) {
		// Simulate API call
		time.Sleep(100 * time.Millisecond)
		
		if id <= 0 {
			return User{}, errors.New("invalid user ID")
		}
		
		return User{
			ID:   id,
			Name: fmt.Sprintf("User %d", id),
		}, nil
	})
}

type User struct {
	ID   int
	Name string
}

type UserWithPosts struct {
	User  User
	Posts []string
}

// Composing multiple async operations
func getUserWithPosts(userID int) *async.Future[UserWithPosts] {
	return async.Then(
		fetchUserData(userID),
		func(user User) *async.Future[UserWithPosts] {
			return async.Go(func() (UserWithPosts, error) {
				// Fetch posts for user
				posts := []string{
					"Post 1 by " + user.Name,
					"Post 2 by " + user.Name,
				}
				
				return UserWithPosts{
					User:  user,
					Posts: posts,
				}, nil
			})
		},
	)
}
