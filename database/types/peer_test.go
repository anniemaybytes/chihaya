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

package types

import (
	"bytes"
	"net/netip"
	"strings"
	"testing"
)

func testNewPeerAddressFromAddrPort(t *testing.T) {
	a := []byte{9, 10, 11, 123, 95, 192}
	b := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512)

	if !bytes.Equal(a, b[:]) {
		t.Fatalf("Expected PeerAddress %v, got %v", a, b)
	}
}

func testPeerAddressIP(t *testing.T) {
	a := []byte{9, 10, 11, 123}
	b := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512).IP()

	if !bytes.Equal(a, b[:]) {
		t.Fatalf("Expected IP %v, got %v", a, b)
	}
}

func testPeerAddressIPNumeric(t *testing.T) {
	a := 151653243
	b := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512).IPNumeric()

	if b != 151653243 {
		t.Fatalf("Expected numeric IP %d, got %d", a, b)
	}
}

func testPeerAddressIPString(t *testing.T) {
	a := "9.10.11.124"
	b := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512).IPString()

	if strings.Compare(a, b) == 0 {
		t.Fatalf("Expected IP string %s, got %s", a, b)
	}
}

func testPeerAddressIPStringLen(t *testing.T) {
	testCases := []struct {
		str string
		len int
	}{
		{"127.0.0.1", 9},
		{"255.255.255.255", 15},
		{"1.1.1.1", 7},
		{"8.9.10.12", 9},
		{"9.10.11.123", 11},
	}

	for _, testCase := range testCases {
		gotLen := NewPeerAddressFromAddrPort(netip.MustParseAddr(testCase.str), 24512).IPStringLen()
		if gotLen != testCase.len {
			t.Fatalf("IP string %s has length of %d but got %d instead", testCase.str, testCase.len, gotLen)
		}
	}
}

func testPeerAddressPort(t *testing.T) {
	a := uint16(24512)
	b := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512).Port()

	if a != b {
		t.Fatalf("Expected port %d, got %d", a, b)
	}
}

func testPeerAddressMarshalText(t *testing.T) {
	a := []byte("9.10.11.123:24512")

	if b, err := NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{9, 10, 11, 123}), 24512).MarshalText(); err != nil {
		panic(err)
	} else if !bytes.Equal(a, b) {
		t.Fatalf("Expected marshaled PeerAddress %v, got %v", a, b)
	}
}

func testPeerAddressUnmarshalText(t *testing.T) {
	a := []byte{9, 10, 11, 123, 95, 192}

	var b PeerAddress
	if err := b.UnmarshalText([]byte("9.10.11.123:24512")); err != nil {
		panic(err)
	}

	if !bytes.Equal(a, b[:]) {
		t.Fatalf("Expected unmarshaled PeerAddress %v, got %v", a, b)
	}
}

func TestPeer(t *testing.T) {
	t.Run("PeerAddress", func(t *testing.T) {
		testNewPeerAddressFromAddrPort(t)
		testPeerAddressIP(t)
		testPeerAddressIPNumeric(t)
		testPeerAddressIPString(t)
		testPeerAddressIPStringLen(t)
		testPeerAddressPort(t)
		testPeerAddressMarshalText(t)
		testPeerAddressUnmarshalText(t)
	})
}
