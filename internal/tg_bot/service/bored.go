package service

import (
	"math/rand/v2"
)

type BoringAPI struct {
	activities []string
}

func NewBoringAPI(activities []string) *BoringAPI {
	return &BoringAPI{
		activities: activities,
	}
}

func (b *BoringAPI) BoredAPI() string {
	n := rand.IntN(len(b.activities))
	return b.activities[n]
}
