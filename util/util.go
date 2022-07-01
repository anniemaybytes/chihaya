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
	"crypto/rand"
	"encoding/binary"
)

const alphanumBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func Min(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func Max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func Btoa(a bool) string {
	if a {
		return "1"
	}

	return "0"
}

func Intn(n int) int {
	b := make([]byte, 8)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	i := binary.BigEndian.Uint32(b)

	return int(i) % n
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphanumBytes[Intn(len(alphanumBytes))]
	}

	return string(b)
}

func Rand(min int, max int) int {
	return Intn(max-min+1) + min
}
