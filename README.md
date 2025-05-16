# Gonoerr

A lightweight Go library providing functional programming patterns for safer error and nil handling.

[![Go Reference](https://pkg.go.dev/badge/github.com/Cyber-cicco/gonoerr.svg)](https://pkg.go.dev/github.com/Cyber-cicco/gonoerr)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cyber-cicco/opt)](https://goreportcard.com/report/github.com/Cyber-cicco/gonoerr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

Opt provides two core types to make Go code more robust and expressive:

- **Result**: A container for operations that might fail, inspired by Rust's Result type
- **Optional**: A container for values that might be nil, inspired by Java's Optional type

Both utilities leverage Go's generics system to provide type-safe, chainable operations that reduce boilerplate and improve error handling.

## Installation

```bash
go get github.com/Cyber-cicco/gonoerr
```

## Usage

### Result

The `Result[T]` type represents either a successful value or an error. This helps avoid panic/recover and provides functional ways to handle errors.

```go
package main

import (
    "errors"
    "fmt"
    "github.com/Cyber-cicco/opt/result"
)

func main() {
    // Create a successful Result
    successResult := result.From("success", nil)
    
    // Create a failed Result
    failedResult := result.From("", errors.New("operation failed"))
    
    // Use the Match method to handle both cases
    successResult.Match(
        func(val string) {
            fmt.Println("Success:", val)
        },
        func(err error) {
            fmt.Println("Error:", err)
        },
    )
    
    // Chain operations
    transformedResult := result.Map(successResult, func(s string) int {
        return len(s)
    })
    
    // Safely unwrap or provide a default
    length := 0
    transformedResult.Match(
        func(val int) { length = val },
        func(err error) { /* error already handled */ },
    )
    
    fmt.Println("Length:", length)
}
```

### Optional

The `Opt[T]` type safely wraps values that might be nil without resorting to pointer checks.

```go
package main

import (
    "fmt"
    "github.com/Cyber-cicco/opt/optional"
)

func main() {
    // Create an Optional with a value
    optWithValue := optional.Of("hello")
    
    // Create an Optional from a nullable pointer
    var nullableString *string
    optEmpty := optional.OfNullable(nullableString)
    
    // Use if present
    optWithValue.IfPresent(func(s *string) {
        fmt.Println("Value:", *s)
    })
    
    // Chain operations with safe handling
    result := optWithValue.
        Filter(func(s *string) bool { return len(*s) > 3 }).
        IfPresent(func(s *string) {
            fmt.Println("Filtered value exists:", *s)
        }).
        OrElseGet("default value")
    
    fmt.Println("Result:", result)
}
```

## API Reference

### Result

```go
// Create a Result from a value and error
result.From[T](ok T, err error) Result[T]

// Methods
(Result[T]) IsOk() bool
(Result[T]) IsErr() bool
(Result[T]) AndThen(fn func(T)) Result[T]
(Result[T]) IfErr(fn func(error)) Result[T]
(Result[T]) Match(ok func(T), err func(error)) Result[T]
(Result[T]) MapErr(fn func(error) error) Result[T]
(Result[T]) UnsafeUnwrap() T  // Panics on error

// Functions
Map[T, V](Result[T], func(T) V) Result[V]
FlatMap[T, V](Result[T], func(T) Result[V]) Result[V]
```

### Optional

```go
// Create an Optional with a value
optional.Of[T](val T) Opt[T]

// Create an Optional from a nullable pointer
optional.OfNullable[T](ptr *T) Opt[T]

// Methods
(Opt[T]) IsPresent() bool
(Opt[T]) IfPresent(fn func(*T)) Opt[T]
(Opt[T]) IfPresentOrElse(present func(*T), absent func()) Opt[T]
(Opt[T]) OrElseDo(fn func()) Opt[T]
(Opt[T]) OrElseGet(val T) T
(Opt[T]) Filter(fn func(*T) bool) Opt[T]
(Opt[T]) AsResult(err error) Result[*T]

// Functions
Map[T, V](Opt[T], func(*T) V) Opt[V]
FlatMap[T, V](Opt[T], func(*T) Opt[V]) Opt[V]
```

## Why use Opt?

- **Improved error handling**: Chain operations safely without extensive error checking
- **Nil safety**: Avoid nil pointer panics by using the Optional pattern
- **Expressiveness**: Write cleaner, more declarative code
- **Functional style**: Leverage Map, FlatMap, and other functional programming patterns
- **Composition**: Build complex logic by composing simple operations

## Design Philosophy

This library was made to implement a simpler way to handle nil values and errors in channels, providing:

1. Type safety through generics
2. Chainable operations for fluent code
3. Functional programming patterns to reduce boilerplate
4. Zero dependencies (except for testing)

## Requirements

- Go 1.18 or later (requires generics support)

## License

MIT License

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.
