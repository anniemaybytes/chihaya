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

package log

import (
	"log"
	"os"
	"runtime/debug"
)

var flags = log.Ldate | log.Ltime | log.LUTC | log.Lmsgprefix

var (
	Verbose = log.New(os.Stdout, "[V] ", flags)
	Info    = log.New(os.Stdout, "[I] ", flags)
	Warning = log.New(os.Stderr, "[W] ", flags)
	Error   = log.New(os.Stderr, "[E] ", flags)
	Fatal   = log.New(os.Stderr, "[F] ", flags)
	Panic   = log.New(os.Stderr, "[P] ", flags)
)

func WriteStack() {
	debug.PrintStack()
}
