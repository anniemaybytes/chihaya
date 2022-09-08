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
	"context"
)

type Semaphore chan struct{}

func NewSemaphore() (s Semaphore) {
	s = make(Semaphore, 1)
	s <- struct{}{}

	return
}

func TakeSemaphore(s Semaphore) {
	<-s
}

func TryTakeSemaphore(ctx context.Context, s Semaphore) bool {
	select {
	case <-s:
		return true
	case <-ctx.Done():
		return false
	}
}

func ReturnSemaphore(s Semaphore) {
	select {
	case s <- struct{}{}:
		return
	default:
		panic("Attempting to return semaphore to an already full channel")
	}
}
