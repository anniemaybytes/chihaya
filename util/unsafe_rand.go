package util

import (
	unsafeRandom "math/rand"
	"time"
)

var randomSource = unsafeRandom.New(unsafeRandom.NewSource(time.Now().Unix()))

func UnsafeInt() int {
	return randomSource.Int()
}

func UnsafeIntn(n int) int {
	return randomSource.Intn(n)
}

func UnsafeUint32() uint32 {
	return randomSource.Uint32()
}

func UnsafeUint64() uint64 {
	return randomSource.Uint64()
}

func UnsafeRand(min int, max int) int {
	return randomSource.Intn(max-min+1) + min
}

func UnsafeReadRand(p []byte) (n int, err error) {
	return randomSource.Read(p)
}
