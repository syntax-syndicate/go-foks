// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type Rational struct {
	proto.Rational
}

type RationalRange struct {
	proto.RationalRange
}

func NewRational(b []byte, exp int64) Rational {
	return Rational{proto.Rational{Base: b, Exp: exp}}
}

func NewRationalInfinity() Rational {
	return Rational{proto.Rational{Infinity: true}}
}

func (r Rational) MaxPow() int {
	if r.Infinity {
		return 0
	}
	return len(r.Base) + int(r.Exp)
}

func (r Rational) MinPow() int {
	if r.Infinity {
		return 0
	}
	return int(r.Exp)
}

func (r Rational) PlaceValue(i int) int {
	if r.Infinity {
		return 0
	}
	mp := r.MaxPow()
	pos := mp - i
	if pos < 0 {
		return 0
	}
	if pos >= len(r.Base) {
		return 0
	}
	return int(r.Base[pos])
}

// Cmp returns 1 if r1 > r2, 0 if r1 == r2, and -1 if r1 < r2
func (r1 Rational) Cmp(r2 Rational) int {
	if r1.Infinity && r2.Infinity {
		return 0
	}
	if r1.Infinity {
		return 1
	}
	if r2.Infinity {
		return -1
	}
	min := min(r1.MinPow(), r2.MinPow())
	max := max(r1.MaxPow(), r2.MaxPow())

	for i := max; i >= min; i-- {
		v1 := r1.PlaceValue(i)
		v2 := r2.PlaceValue(i)
		if v1 > v2 {
			return 1
		}
		if v1 < v2 {
			return -1
		}
	}
	return 0
}

func (r Rational) String() string {
	if r.Infinity {
		return "∞"
	}
	scratch := append([]byte{}, r.Base...)

	if r.Exp < 0 && int(-r.Exp) > len(scratch) {
		diff := int(-r.Exp) - len(scratch)
		scratch = append(make([]byte, diff), scratch...)
	} else if r.Exp > 0 {
		scratch = append(scratch, make([]byte, r.Exp)...)
	}

	decPointAt := len(scratch) + int(r.Exp)
	var intPart []byte
	var fracPart []byte

	if decPointAt < 0 {
		panic("decPointAt < 0")
	}
	if decPointAt < len(scratch) {
		intPart = scratch[:decPointAt]
		fracPart = scratch[decPointAt:]
	} else {
		intPart = scratch
	}

	for len(fracPart) > 0 && fracPart[len(fracPart)-1] == 0 {
		fracPart = fracPart[:len(fracPart)-1]
	}
	for len(intPart) > 0 && intPart[0] == 0 {
		intPart = intPart[1:]
	}

	toString := func(b []byte) string {
		bytes := make([]string, len(b))
		for i, b := range b {
			bytes[i] = fmt.Sprintf("%02x", b)
		}
		return strings.Join(bytes, "")
	}

	if len(intPart) == 0 && len(fracPart) == 0 {
		return "00"
	}

	ret := toString(intPart)
	if len(fracPart) > 0 {
		ret += "." + toString(fracPart)
	}
	return ret
}

func ParseRational(s string) (*Rational, error) {
	s = strings.TrimSpace(s)
	if s == "∞" {
		tmp := NewRationalInfinity()
		return &tmp, nil
	}

	decPointAt := strings.Index(s, ".")
	var exp int

	if decPointAt >= 0 {
		if decPointAt%2 != 0 || len(s)%2 != 1 {
			return nil, BadFormatError("rational must have an even number of digits")
		}
		exp = (decPointAt - len(s) + 1) / 2
		s = strings.Replace(s, ".", "", 1)
	} else if len(s)%2 != 0 {
		return nil, BadFormatError("rational must have an even number of digits")
	}
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	for len(bytes) > 0 && bytes[0] == 0 {
		bytes = bytes[1:]
	}
	for len(bytes) > 0 && bytes[len(bytes)-1] == 0 {
		bytes = bytes[:len(bytes)-1]
		exp++
	}
	ret := NewRational(bytes, int64(exp))
	return &ret, nil
}

func (r RationalRange) GetLow() Rational  { return Rational{r.Low} }
func (r RationalRange) GetHigh() Rational { return Rational{r.High} }

func (r RationalRange) Validate() error {
	err := r.Low.Validate()
	if err != nil {
		return err
	}
	err = r.High.Validate()
	if err != nil {
		return err
	}
	if r.GetLow().Cmp(r.GetHigh()) > 0 {
		return BadRangeError{}
	}
	return nil
}

func (r RationalRange) SetHigh(r2 Rational) RationalRange {
	r.High = r2.Rational
	return r
}

func (r RationalRange) SetLow(r2 Rational) RationalRange {
	r.Low = r2.Rational
	return r
}

func (r1 RationalRange) LessThan(r2 RationalRange) bool {
	return r1.GetHigh().Cmp(r2.GetLow()) < 0
}

func (r1 RationalRange) Eq(r2 RationalRange) bool {
	return r1.GetLow().Cmp(r2.GetLow()) == 0 && r1.GetHigh().Cmp(r2.GetHigh()) == 0
}

func (r1 RationalRange) Includes(r2 RationalRange) bool {
	return r1.GetLow().Cmp(r2.GetLow()) <= 0 && r1.GetHigh().Cmp(r2.GetHigh()) >= 0
}

func (r Rational) Rsh() (*Rational, error) {
	if r.Infinity {
		return nil, BadArgsError("∞ cannot be shifted")
	}

	newBase := make([]byte, len(r.Base))
	newExp := r.Exp

	carry := false
	for i, b := range r.Base {
		var newCarry bool
		if b&0x01 != 0 {
			newCarry = true
		}
		b = b >> 1
		if carry {
			b |= 0x80
		}
		newBase[i] = b
		carry = newCarry
	}
	if carry {
		newBase = append(newBase, 0x80)
		newExp--
	}
	// We might have shifted off leading 0s
	if len(newBase) > 0 && newBase[0] == 0 {
		newBase = newBase[1:]
	}
	return &Rational{proto.Rational{Base: newBase, Exp: newExp}}, nil
}

func (r Rational) Lsh() (*Rational, error) {
	if r.Infinity {
		return nil, BadArgsError("∞ cannot be raised")
	}

	carry := false
	newBase := make([]byte, len(r.Base))
	for i := len(r.Base) - 1; i >= 0; i-- {
		b := r.Base[i]
		var newCarry bool
		if b&0x80 != 0 {
			newCarry = true
		}
		b = b << 1
		if carry {
			b |= 0x01
		}
		newBase[i] = b
		carry = newCarry
	}
	if carry {
		newBase = append([]byte{0x01}, newBase...)
	}
	exp := r.Exp
	l := len(newBase) - 1
	if len(newBase) > 0 && newBase[l] == 0 {
		newBase = newBase[:l]
		exp++
	}

	ret := Rational{proto.Rational{Base: newBase, Exp: exp}}
	return &ret, nil
}

func (r RationalRange) Rsh() (*RationalRange, error) {
	high, err := r.GetHigh().Rsh()
	if err != nil {
		return nil, err
	}
	r.High = high.Rational
	return &r, nil
}

func (r RationalRange) Lsh() (*RationalRange, error) {
	low, err := r.GetLow().Lsh()
	if err != nil {
		return nil, err
	}
	r.Low = low.Rational
	return &r, nil
}

func ParseRationalRange(s string) (*RationalRange, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		s = s[1 : len(s)-1]
	}
	rxx := regexp.MustCompile(`[-,]`)
	parts := rxx.Split(s, -1)
	if len(parts) != 2 {
		return nil, BadFormatError("range must have exactly one '-'")
	}
	low, err := ParseRational(parts[0])
	if err != nil {
		return nil, err
	}
	highStr := Sel(len(parts[1]) == 0, "∞", parts[1])
	high, err := ParseRational(highStr)
	if err != nil {
		return nil, err
	}
	ret := RationalRange{proto.RationalRange{Low: low.Rational, High: high.Rational}}
	err = ret.Validate()
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func NewDefaultRange() RationalRange {
	return RationalRange{proto.RationalRange{
		Low:  proto.Rational{Base: []byte{0x01}, Exp: 0},
		High: proto.Rational{Infinity: true},
	}}
}

func NewRationalRange(r proto.RationalRange) RationalRange {
	return RationalRange{r}
}

func (r RationalRange) String() string {
	return fmt.Sprintf("%s-%s", r.GetLow().String(), r.GetHigh().String())
}

func (r RationalRange) StringParen() string {
	return fmt.Sprintf("(%s)", r.String())
}

func (r RationalRange) Export() proto.RationalRange {
	return r.RationalRange
}

func (r *RationalRange) ExportP() *proto.RationalRange {
	if r == nil {
		return nil
	}
	tmp := r.Export()
	return &tmp
}
