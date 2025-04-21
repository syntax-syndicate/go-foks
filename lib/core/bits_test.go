// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUppermosteBit(t *testing.T) {
	for i := 0; i < 0x100; i++ {
		res := UppermostBitSet(byte(i))
		var e int
		if i < 1 {
			e = -1
		} else if i < 0x2 {
			e = 0
		} else if i < 0x4 {
			e = 1
		} else if i < 0x8 {
			e = 2
		} else if i < 0x10 {
			e = 3
		} else if i < 0x20 {
			e = 4
		} else if i < 0x40 {
			e = 5
		} else if i < 0x80 {
			e = 6
		} else {
			e = 7
		}
		require.Equal(t, e, res)
	}
}

func TestBitAt(t *testing.T) {

	var tests = []struct {
		key []byte
		loc int
		res bool
		err error
	}{
		{[]byte{0x80}, 0, true, nil},
		{[]byte{0x40}, 0, false, nil},
		{[]byte{0x4f}, 1, true, nil},
		{[]byte{0x4f, 0x4f, 0x4f}, 8, false, nil},
		{[]byte{0x4f, 0x4f, 0x4f}, 9, true, nil},
		{[]byte{0x4f, 0x4f, 0x4f}, 12, true, nil},
		{[]byte{0x4f, 0x4f, 0x4f}, 30, false, InternalError("bit location is out of range")},
	}

	for _, test := range tests {
		res, err := BitAt(test.key, test.loc)
		require.Equal(t, res, test.res)
		require.Equal(t, err, test.err)
	}

}

// "1111 0000" -> (0xf0, 8)
// "1111" ->      (0xf0, 4)
// "1111 1111 11" -> ([0xff, 0xb0], 10)
func convertBinary(t *testing.T, s string) ([]byte, int) {
	s = strings.ReplaceAll(s, " ", "")
	bitPos := 7
	res := []byte{0}
	bytePos := 0
	for _, ch := range []byte(s) {

		if bitPos < 0 {
			res = append(res, byte(0))
			bytePos++
			bitPos = 7
		}

		switch ch {
		case '0':
		case '1':
			res[bytePos] |= (1 << bitPos)
		default:
			t.Error("got bad char in binary string")
		}

		bitPos--
	}
	return res, len(s)
}

func TestConvertBinary(t *testing.T) {
	var tests = []struct {
		s string
		b []byte
		l int
	}{
		{"11", []byte{0xc0}, 2},
		{"1111 1111 0000 0000", []byte{0xff, 0x00}, 16},
		{"1111 1111 0000 00", []byte{0xff, 0x00}, 14},
		{"1100 1100 0011 11", []byte{0xcc, 0x3c}, 14},
	}

	for _, test := range tests {
		res, l := convertBinary(t, test.s)
		require.Equal(t, test.b, res)
		require.Equal(t, test.l, l)
	}
}

func TestAssertKeyMatch(t *testing.T) {

	conv := func(s string) []byte {
		ret, _ := convertBinary(t, s)
		return ret
	}

	var tests = []struct {
		key     []byte
		segment []byte
		start   int
		count   int
		res     error
	}{
		{conv("1101 1001 0110 1010"), conv("0001 10"), 2, 4, nil},
		{conv("1101 1001 0110 1010"), conv("0001 11"), 2, 4, BitPrefixMatchError(5)},
		{conv("1101 1001 0110 1010"), conv("011"), 8, 3, nil},
		{conv("1101 1001 0110 1010"), conv("001"), 8, 3, BitPrefixMatchError(9)},
		{conv("1101 1001 0110 1010 0011 1010 10101"), conv("0011 1"), 17, 4, nil},
		{conv("1101 1001 0110 1010 0011 1010 10101"), conv("1"), 0, 0, nil},
		{conv("1101 1001 0110 1010 0011 1010 10101"), conv("0"), 0, 1, BitPrefixMatchError(0)},
		{conv("1101 1001 0110 1010 0011 1010 10101"), conv("1101 1001"), 0, 7, nil},
		{conv("1101 1001 0110 1010 0011 1010 10101"), conv("0001 1001 0110 1010 0011 1"), 2, 19, nil},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 0, 1, BitPrefixMatchError(0)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 1, 1, BitPrefixMatchError(1)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 2, 1, BitPrefixMatchError(2)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 3, 1, BitPrefixMatchError(3)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 4, 1, BitPrefixMatchError(4)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 5, 1, BitPrefixMatchError(5)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 6, 1, BitPrefixMatchError(6)},
		{conv("1111 1111 0110 1010"), conv("0000 0000"), 6, 0, nil},
	}

	for _, test := range tests {
		res := AssertKeyMatch(test.key, test.segment, test.start, test.count)
		require.Equal(t, test.res, res)
	}
}

func TestComputePrefixMatch(t *testing.T) {

	conv := func(s string) []byte {
		ret, _ := convertBinary(t, s)
		return ret
	}

	var tests = []struct {
		x           []byte
		y           []byte
		bitsMatched int
		res         []byte
		resLen      int
		resSplit    bool
		err         error
	}{
		{
			conv("1101 0111 1010 0000 1111 1010 0010"),
			conv("1101 0111 1010 0000 1111 1010 0000"),
			9,
			conv("1010 0000 1111 1010 0010"),
			17,
			true,
			nil,
		},
		{conv("1100"), conv("1000"), 0, conv("1100 0000"), 1, true, nil},
		{conv("1100"), conv("1100"), 0, nil, 0, false, InternalError("did not expect x=y in ComputePrefixMatch")},
		{
			conv("0101 1010 0000 1111 1100 0011 1001 0110"),
			conv("0101 1010 0000 1111 1000 0011 1001 0110"),
			10,
			conv("0000 1111 1100 0011"),
			7,
			true,
			nil,
		},
		{
			conv("0101 1010 0000 1111 1000 0011 1001 0110"),
			conv("0101 1010 0000 1111 1000 0011 1001 0111"),
			1,
			conv("0101 1010 0000 1111 1000 0011 1001 0110"),
			30,
			false,
			nil,
		},
	}
	for _, tst := range tests {
		res, resLen, resSplit, err := ComputePrefixMatch(tst.x, tst.y, tst.bitsMatched)
		require.Equal(t, tst.res, res)
		require.Equal(t, tst.resLen, resLen)
		require.Equal(t, tst.resSplit, resSplit)
		require.Equal(t, tst.err, err)
	}

}

func TestShiftCopyAndClamp(t *testing.T) {

	var tests = []struct {
		in     string
		start  int
		count  int
		out    string
		nBytes int
	}{
		{
			"0011 1110",
			3,
			2,
			"0001 1000",
			0,
		},
		{
			"0011 1110",
			67,
			2,
			"0001 1000",
			0,
		},
		{
			"0011 1101 1010 1111 0101 0101",
			3,
			11,
			"0001 1101 1010 1100",
			1,
		},
		{
			"1111 1111 1111 1111 1111 1111 1111 1111",
			3,
			7,
			"0001 1111 1100 0000",
			1,
		},
		{
			"1111 1111",
			3,
			5,
			"0001 1111",
			1,
		},
	}

	for _, tst := range tests {
		inB, _ := convertBinary(t, tst.in)
		expected, _ := convertBinary(t, tst.out)
		out, nBytes := ShiftCopyAndClamp(inB, tst.start, tst.count)
		require.Equal(t, expected, out)
		require.Equal(t, tst.nBytes, nBytes)
	}
}

func TestCopyAndClamp(t *testing.T) {

	var tests = []struct {
		in    string
		start int
		count int
		out   string
	}{
		{
			"0011 1110",
			3,
			2,
			"0001 1000",
		},
		{
			"0011 1101 1010 1111 0101 0101",
			3,
			11,
			"0001 1101 1010 1100",
		},
		{
			"1111 1111 1111 1111 1111 1111 1111 1111",
			3,
			7,
			"0001 1111 1100 0000",
		},
		{
			"1111 1111",
			3,
			5,
			"0001 1111",
		},
		{
			"0101 01010 1010 1010 1111 1111",
			19,
			4,
			"0001 1110",
		},
	}

	for _, tst := range tests {
		inB, _ := convertBinary(t, tst.in)
		expected, _ := convertBinary(t, tst.out)
		out := CopyAndClamp(inB, tst.start, tst.count)
		require.Equal(t, expected, out)
	}
}
