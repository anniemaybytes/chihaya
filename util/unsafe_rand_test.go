package util

import (
	"math/rand"
	"testing"
)

func TestUnsafeRand(t *testing.T) {
	for i := 0; i < 10; i++ {
		min := rand.Intn(1000)
		max := rand.Intn(1000) + min
		randomInt := UnsafeRand(min, max)

		if randomInt < min || randomInt > max {
			t.Fatalf("Integer %d is outside specified range (%d - %d)", randomInt, min, max)
		}
	}
}

func TestUnsafeIntn(t *testing.T) {
	for i := 1; i < 2000; i++ {
		genInt := UnsafeIntn(i)

		if genInt < 0 || genInt >= i {
			t.Fatalf("Generated random integer (%d) does not fall in the range [0, %d)!", genInt, i)
		}
	}
}
