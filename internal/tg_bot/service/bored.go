// Package service provides a simple implementation for generating activity suggestions.
// It includes a BoringAPI struct that randomly selects activities from a provided list.
package service

import (
	"math/rand/v2"
	"time"
)

// BoringAPI manages a list of activity suggestions and provides random selections.
type BoringAPI struct {
	activities []string
	rng        *rand.Rand
}

// NewBoringAPI creates a new instance of BoringAPI with the specified activities.
// Arguments:
//   - activities: a slice of strings representing possible activity suggestions.
//
// Returns a pointer to a BoringAPI.
func NewBoringAPI(activities []string) *BoringAPI {
	if len(activities) == 0 {
		return nil // Or handle differently based on requirements
	}
	return &BoringAPI{
		activities: activities,
		rng:        rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)), // Seed with time
	}
}

// BoredAPI returns a randomly selected activity from the list.
// Returns a string representing the selected activity.
func (b *BoringAPI) BoredAPI() string {
	if len(b.activities) == 0 {
		return "" // Or return an error if the method signature changes
	}
	n := b.rng.IntN(len(b.activities))
	return "Ты можешь " + b.activities[n]
}
