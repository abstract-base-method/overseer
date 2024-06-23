package common

import (
	"math/rand"
	"time"
)

func Reduce[T any](s []T, fn func(T) bool) []T {
	var p []T
	for _, v := range s {
		if fn(v) {
			p = append(p, v)
		}
	}
	return p
}

func Filter[T any, O any](input []T, fn func(T) O) []O {
	var p []O
	for _, v := range input {
		p = append(p, fn(v))
	}
	return p
}

func FindFirst[T any](s []T, fn func(T) bool) *T {
	for _, v := range s {
		if fn(v) {
			return &v
		}
	}
	return nil
}

func RandomizedProgressiveValue(minimum float32, selection float32, maximum float32, maxInt int) int {
	// Ensure selection is within the range
	if selection < minimum {
		selection = minimum
	} else if selection > maximum {
		selection = maximum
	}

	// Calculate the normalized position of selection between minimum and maximum
	normalized := (selection - minimum) / (maximum - minimum)

	// Generate a random value in the range [0, 1) and multiply by normalized position
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomFactor := r.Float32() * normalized

	// Scale the random factor to the maxInt range and ensure the result is nonzero
	result := int(randomFactor * float32(maxInt))
	if result == 0 {
		result = 1
	}

	return result
}
