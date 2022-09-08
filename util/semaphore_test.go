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
	"testing"
	"time"
)

func TestTakeReturnSemaphore(t *testing.T) {
	var (
		testSemaphore  = NewSemaphore()
		panickedOnFull = false
	)

	TakeSemaphore(testSemaphore)

	if len(testSemaphore) != 0 {
		t.Fatalf(
			"Semaphore channel length incorrect after consuming synchronously: is %v but should be 0",
			len(testSemaphore))
	}

	ReturnSemaphore(testSemaphore)

	if len(testSemaphore) != 1 {
		t.Fatalf(
			"Semaphore channel length incorrect after returning synchronously: is %v but should be 1",
			len(testSemaphore))
	}

	defer func() {
		if r := recover(); r != nil {
			panickedOnFull = true
		}
	}()

	ReturnSemaphore(testSemaphore)

	if !panickedOnFull {
		t.Fatalf("ReturnSemaphore must panic when attempting to return to an already full channel")
	}
}

func TestTryTakeSemaphore(t *testing.T) {
	var (
		testSemaphore = NewSemaphore()

		ctx        context.Context
		cancelFunc context.CancelFunc
	)

	ctx, cancelFunc = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelFunc()

	if !TryTakeSemaphore(ctx, testSemaphore) {
		t.Fatalf("Failed to consume semaphore: %v; channel length at %v", ctx.Err(), len(testSemaphore))
	}

	if len(testSemaphore) != 0 {
		t.Fatalf(
			"Semaphore channel length incorrect after taking asynchronously: is %v but should be 0",
			len(testSemaphore))
	}
}
