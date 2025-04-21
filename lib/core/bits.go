// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"bytes"
)

// UpperMostBitSet returns the upper most bitset in the byte, where
// 0x80 returns 7 and 0x1 returns 0. Should not pass 0x0 but if you
// do, you'll get back -1.
func UppermostBitSet(b byte) int {
	if b&0xf0 != 0 { // 1111 0000
		if b&0xc0 != 0 { // 1100 0000
			if b&0x80 != 0 { // 1000 0000
				return 7
			} else {
				return 6
			}
		} else {
			if b&0x20 != 0 { // 0010 0000
				return 5
			} else {
				return 4
			}
		}
	} else {
		if b&0xc != 0 { // 0000 1100
			if b&0x8 != 0 { // 0000 1000
				return 3
			} else {
				return 2
			}
		} else {
			if b&0x2 != 0 { // 0000 0010
				return 1
			} else if b == 0x1 { //
				return 0
			}
		}
	}
	return -1
}

func AssertKeyMatch(key []byte, sub []byte, start int, count int) error {
	if count == 0 {
		return nil
	}
	startByte := start >> 3
	end := start + count - 1
	endByte := end >> 3
	j := 0
	for i := startByte; i <= endByte && j < len(sub); i++ {
		if i >= len(key) {
			return InternalError("fell off end of key string")
		}

		topMask := byte(0xff)
		if i == startByte {
			// now map startBitOffset 3 -> 00011111
			topMask >>= (start & 0x7)
		}

		bottomMask := byte(0xff)
		if i == endByte {
			// now map endBitOffset 6 -> 1111 1100
			bottomMask <<= (7 - (end & 0x7))
		}

		tmp := (key[i] ^ sub[j]) & topMask & bottomMask
		if tmp != 0 {
			fbs := UppermostBitSet(tmp)
			if fbs < 0 {
				return InternalError("didn't find a set bit")
			}
			return BitPrefixMatchError((i << 3) + (7 - fbs))
		}

		j++
	}
	return nil
}

// Given two keys --- x & y --- that have already matched bitsMatched bits
// via tree traversal, compute the longest string starting at bitsMatched
// that is common to both. Return the prefix and the number of bits matched, and also
// the bit value of x at the first mismatch.
func ComputePrefixMatch(x []byte, y []byte, bitsMatched int) ([]byte, int, bool, error) {

	if len(x) != len(y) {
		return nil, 0, false, InternalError("inputs must have equal length")
	}
	if bytes.Equal(x, y) {
		return nil, 0, false, InternalError("did not expect x=y in ComputePrefixMatch")
	}

	startByte := bitsMatched >> 3
	if startByte >= len(x) {
		return nil, 0, false, InternalError("bitsMatched was too big")
	}

	x = x[startByte:]
	y = y[startByte:]

	newBitsMatched := 0
	firstXMismatchBit := false

	for i, c := range x {
		d := y[i]
		if c == d {
			newBitsMatched += 8
		} else {
			pos := UppermostBitSet(c ^ d)
			firstXMismatchBit = (c & (1 << pos)) != 0
			newBitsMatched += (7 - pos)
			break
		}
	}

	rightFence := ((newBitsMatched + 7) >> 3)

	ret := x[0:rightFence]

	// Don't give credit for the leading bits in x or y that are already covered in bitsMatched.
	newBitsMatched -= (bitsMatched & 0x7)
	return ret, newBitsMatched, firstXMismatchBit, nil
}

func BitAt(key []byte, bitPos int) (bool, error) {
	byteIndex := bitPos >> 3
	if byteIndex >= len(key) {
		return false, InternalError("bit location is out of range")
	}
	ret := (key[byteIndex] & (1 << (7 - (bitPos & 0x7)))) != 0
	return ret, nil
}

func ShiftCopyAndClamp(in []byte, start int, count int) ([]byte, int) {
	if count == 0 {
		return []byte{}, 0
	}

	// endBit and endByte are inclusive
	startBit := start & 0x7
	endBit := startBit + count - 1
	endByte := endBit >> 3

	buf := make([]byte, endByte+1)
	copy(buf[:], in)
	leftMask := byte(0xff) >> startBit
	rightMask := byte(0xff) << (0x7 - (endBit & 0x7))
	buf[0] &= leftMask
	buf[endByte] &= rightMask
	nBytesUsed := endByte
	if endBit&0x7 == 0x7 {
		nBytesUsed++
	}
	return buf, nBytesUsed
}

func CopyAndClamp(in []byte, start int, count int) []byte {
	if count == 0 {
		return []byte{}
	}

	// endBit and endByte are inclusive
	startByte := start >> 3
	startBit := start
	endBit := startBit + count - 1
	endByte := endBit >> 3

	buf := make([]byte, endByte-startByte+1)
	if endByte >= len(in) {
		return nil
	}
	copy(buf[:], in[startByte:endByte+1])
	leftMask := byte(0xff) >> (startBit & 0x7)
	rightMask := byte(0xff) << (0x7 - (endBit & 0x7))
	buf[0] &= leftMask
	buf[len(buf)-1] &= rightMask
	return buf
}
