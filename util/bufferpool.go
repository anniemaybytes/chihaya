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
	"sync"
)

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool(bufSize int) *BufferPool {
	p := &BufferPool{}
	p.pool.New = func() any {
		internalBuf := make([]byte, 0, bufSize)
		return bytes.NewBuffer(internalBuf)
	}

	return p
}

func (pool *BufferPool) Take() (buf *bytes.Buffer) {
	buf = pool.pool.Get().(*bytes.Buffer)
	buf.Reset()

	return
}

func (pool *BufferPool) Give(buf *bytes.Buffer) {
	pool.pool.Put(buf)
}
