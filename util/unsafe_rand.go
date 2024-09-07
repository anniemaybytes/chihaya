/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

package util

import (
	unsafeRandom "math/rand"
	"sync"
	"time"
)

var randomSourcePool sync.Pool

func init() {
	randomSourcePool.New = func() any {
		return unsafeRandom.New(unsafeRandom.NewSource(time.Now().UnixNano())) //nolint:gosec
	}
}

func UnsafeInt() int {
	randomSource := randomSourcePool.Get().(*unsafeRandom.Rand)
	defer randomSourcePool.Put(randomSource)

	return randomSource.Int()
}

func UnsafeIntn(n int) int {
	randomSource := randomSourcePool.Get().(*unsafeRandom.Rand)
	defer randomSourcePool.Put(randomSource)

	return randomSource.Intn(n)
}

func UnsafeUint32() uint32 {
	randomSource := randomSourcePool.Get().(*unsafeRandom.Rand)
	defer randomSourcePool.Put(randomSource)

	return randomSource.Uint32()
}

func UnsafeUint64() uint64 {
	randomSource := randomSourcePool.Get().(*unsafeRandom.Rand)
	defer randomSourcePool.Put(randomSource)

	return randomSource.Uint64()
}

func UnsafeReadRand(p []byte) (n int, err error) {
	randomSource := randomSourcePool.Get().(*unsafeRandom.Rand)
	defer randomSourcePool.Put(randomSource)

	return randomSource.Read(p)
}
