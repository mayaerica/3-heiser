package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/fallback"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"
)

// function which calculate x+1 and failed randomly
func UnreliableFunction(x int) (int, error) {
	if rand.Intn(2) == 0 {
		return 0, errors.New("échec de connexion")
	}
	return x + 1, nil
}

// Function with random delay
func SlowFunction() error {
	duration := time.Duration(rand.Intn(3)) * time.Second
	time.Sleep(duration)
	if duration > 2*time.Second {
		return errors.New("timeout dépassé")
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// ✅ Retry Policy: Try 3 times if errors
	retryPolicy := retrypolicy.Builder[any]().
		HandleErrors(errors.New("échec de connexion")).
		WithMaxRetries(3).
		Build()

	// ✅ Fallback Policy: use a default value if error
	fallbackPolicy := fallback.WithResult("Fallback result")

	// ✅ Circuit Breaker: stop after 3 errors one after the other
	cb := circuitbreaker.WithThreshold(5) // Timeout Policy: Annule après 2 secondes
	timeoutPolicy := timeout.With(2 * time.Second)

	// Execution avec Retry
	fmt.Println("Test Retry:")
	result, err := failsafe.Run(UnreliableFunction, retryPolicy, 5)  // Utilisation de la valeur 5 comme exemple
	if err != nil {
		fmt.Println("Final Failure after retries:", err)
	} else {
		fmt.Println("Success after retrying, result:", result)
	}

	// Execution avec Fallback
	fmt.Println("\nTest Fallback:")
	resultFallback := failsafe.Get(UnreliableFunction, fallbackPolicy, 5)
	fmt.Println("Result:", resultFallback)

	// Execution avec Circuit Breaker
	fmt.Println("\nTest Circuit Breaker:")
	for i := 0; i < 5; i++ {
		if err := failsafe.Run(UnreliableFunction, cb, 5); err != nil {
			fmt.Println("Circuit breaker active:", err)
		} else {
			fmt.Println("Success")
		}
	}

	// Execution avec Timeout
	fmt.Println("\nTest Timeout:")
	err = failsafe.Run(SlowFunction, timeoutPolicy)
	if err != nil {
		fmt.Println("Timeout triggered:", err)
	} else {
		fmt.Println("Success within time limit")
	}
}
