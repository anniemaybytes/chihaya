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
	"testing"
)

func TestMin(t *testing.T) {
	for i := 0; i < 100; i++ {
		a := UnsafeInt()
		b := UnsafeInt()
		gotMin := Min(a, b)

		var actualMin int
		if b > a {
			actualMin = a
		} else {
			actualMin = b
		}

		if actualMin != gotMin {
			t.Fatalf("Min value (%d) is wrong for a=%d and b=%d!", gotMin, a, b)
		}
	}
}

func TestMax(t *testing.T) {
	for i := 0; i < 100; i++ {
		a := UnsafeInt()
		b := UnsafeInt()
		gotMax := Max(a, b)

		var actualMax int
		if b < a {
			actualMax = a
		} else {
			actualMax = b
		}

		if actualMax != gotMax {
			t.Fatalf("Max value (%d) is wrong for a=%d and b=%d!", gotMax, a, b)
		}
	}
}

func TestBtoa(t *testing.T) {
	for i := 0; i < 100; i++ {
		var b bool

		var actualResult string

		if UnsafeIntn(2) == 1 {
			b = true
			actualResult = "1"
		} else {
			b = false
			actualResult = "0"
		}

		gotResult := Btoa(b)
		if actualResult != gotResult {
			t.Fatalf("Got wrong string (%s) for bool %t!", gotResult, b)
		}
	}
}
