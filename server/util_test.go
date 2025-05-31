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
	"net/netip"
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

func TestIsPrivateIpAddress(t *testing.T) {
	privateIps := []string{
		"0.0.0.0",
		"127.0.0.2",
		"10.10.10.1",
		"172.18.0.254",
		"192.168.0.125",
		"169.254.69.2",
		"::",
		"::1",
		"fe80:dead:beef::1",
	}

	for _, ipAddr := range privateIps {
		if !isPrivateIPAddress(netip.MustParseAddr(ipAddr)) {
			t.Fatalf("Private IP %s was reported as public", ipAddr)
		}
	}

	publicIps := []string{
		"45.128.19.54",
		"2606:4700:4700::1111",
	}

	for _, ipAddr := range publicIps {
		if isPrivateIPAddress(netip.MustParseAddr(ipAddr)) {
			t.Fatalf("Public IP %s was reported as private", ipAddr)
		}
	}
}
