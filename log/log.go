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
	"runtime/debug"
)

var (
	writer = log.Writer()
	flags  = log.Ldate | log.Ltime | log.LUTC | log.Lmsgprefix
)

var (
	Info    = log.New(writer, "[I] ", flags)
	Warning = log.New(writer, "[W] ", flags)
	Error   = log.New(writer, "[E] ", flags)
	Fatal   = log.New(writer, "[F] ", flags)
	Panic   = log.New(writer, "[P] ", flags)
)

func WriteStack() {
	debug.PrintStack()
}
