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
	"bytes"
	"testing"
)

func TestBufferPool(t *testing.T) {
	bufferPool := NewBufferPool(64)

	poolBuf := bufferPool.Take()
	if !bytes.Equal(poolBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from empty bufferpool was allocated incorrectly.")
	}

	origBuf := bytes.NewBuffer([]byte("X"))
	bufferPool.Give(origBuf)

	reusedBuf := bufferPool.Take()
	if !bytes.Equal(reusedBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from filled bufferpool was recycled incorrectly.")
	}

	if &origBuf.Bytes()[:1][0] != &reusedBuf.Bytes()[:1][0] {
		t.Fatalf("Recycled buffer points at different address.")
	}
}
