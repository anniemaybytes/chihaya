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

package server

import (
	"bytes"
	"testing"
	"time"
)

func TestFailure(t *testing.T) {
	buf := bytes.NewBufferString("some existing data")

	failure("error message", buf, time.Second*5)

	testData := []byte("d14:failure reason13:error message8:intervali5ee")
	if !bytes.Equal(buf.Bytes(), testData) {
		t.Fatalf("Expected %s, got %s", testData, buf.Bytes())
	}
}
