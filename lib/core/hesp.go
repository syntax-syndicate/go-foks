// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type HESPConfig struct {
	numWords    int
	intBits     int
	numInts     int
	intModulus  int
	wordModulus int
	wordBits    int
	totBits     int
	totBytes    int
	topMask     byte
}

func NewHESPConfig(w int, b int) *HESPConfig {
	r := &HESPConfig{
		numWords:    w,
		intBits:     b,
		numInts:     w - 1,
		intModulus:  1 << b,
		wordModulus: len(BIP39List),
	}
	r.wordBits = log2(r.wordModulus)
	r.totBits = r.numWords*r.wordBits + r.numInts*r.intBits
	r.totBytes = (r.totBits + 7) / 8
	topMaskBits := r.totBytes*8 - r.totBits
	r.topMask = byte(0xff) >> byte(topMaskBits)
	return r
}

func (c HESPConfig) Clamp(b []byte) error {
	if len(b) != c.totBytes {
		return HESPError("wrong number of input bytes")
	}
	b[0] &= c.topMask
	return nil
}

func (c HESPConfig) GenerateSecret(b []byte) error {
	n, err := rand.Read(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return HESPError("short read")
	}
	return c.Clamp(b)
}

func (c HESPConfig) ValidateInput(s string) error {
	h := NewHESP(&c)
	return h.FromString(s)
}

func (c HESPConfig) TotalBytes() int {
	return c.totBytes
}

var KexSeedHESPConfig = NewHESPConfig(7, 8) // == 7*11+6*8 = 125 bits of entropy

// HESP = High-entropy secret phrase
type HESP struct {
	words []int
	ints  []int
	raw   []byte
	c     *HESPConfig
}

func NewHESP(c *HESPConfig) *HESP {
	return &HESP{
		c: c,
	}
}
func log2(n int) int {
	ret := 0
	for n > 1 {
		n >>= 1
		ret++
	}
	return ret
}

func (k *HESP) Import(s []byte) error {
	if len(s) != k.c.totBytes {
		return HESPError("wrong number of input bytes")
	}
	k.raw = make([]byte, k.c.totBytes)
	copy(k.raw, s[:])
	return k.fillWords()
}

func (k *HESP) Generate() error {
	topMask := k.c.topMask
	k.raw = make([]byte, k.c.totBytes)
	n, err := rand.Read(k.raw)
	if err != nil {
		return err
	}
	if n != k.c.totBytes {
		return InternalError("short read")
	}
	k.raw[0] &= topMask
	return k.fillWords()
}

func (k *HESP) fillWords() error {

	b := big.NewInt(0).SetBytes(k.raw)

	var wordMask, numMask big.Int
	wordMask.Sub(big.NewInt(int64(k.c.wordModulus)), big.NewInt(1))
	numMask.Sub(big.NewInt(int64(k.c.intModulus)), big.NewInt(1))

	grabWord := func() {
		var tmp big.Int
		n := tmp.And(b, &wordMask).Int64()
		k.words = append(k.words, int(n))
		b.Rsh(b, uint(k.c.wordBits))
	}
	grabInt := func() {
		var tmp big.Int
		n := tmp.And(b, &numMask).Int64()
		k.ints = append(k.ints, int(n))
		b.Rsh(b, uint(k.c.intBits))
	}

	for i := 0; i < k.c.numInts; i++ {
		grabWord()
		grabInt()
	}
	grabWord()

	if b.Sign() != 0 {
		return InternalError("leftover bits")
	}

	return nil
}

func (k HESP) Tokens() []string {
	ret := make([]string, 0, len(k.words)+len(k.ints))
	for i, w := range k.words {
		ret = append(ret, BIP39List[w])
		if i < len(k.ints) {
			ret = append(ret, fmt.Sprintf("%d", k.ints[i]))
		}
	}
	return ret
}

func (k HESP) Export() []string {
	ret := k.Tokens()
	return ret
}

func (k HESP) ToString() string {
	return strings.Join(k.Tokens(), " ")
}

func (k *HESP) FromString(s string) error {
	return k.FromTokens(strings.Fields(s))
}

func (k *HESP) fillRaw() {
	b := &big.Int{}

	words := k.words
	ints := k.ints

	pop := func(v *([]int)) int {
		l := len(*v)
		ret := (*v)[l-1]
		*v = (*v)[:l-1]
		return ret
	}

	grabWord := func() {
		i := pop(&words)
		b = b.Lsh(b, uint(k.c.wordBits))
		b = b.Or(b, big.NewInt(int64(i)))
	}

	grabInt := func() {
		i := pop(&ints)
		b = b.Lsh(b, uint(k.c.intBits))
		b = b.Or(b, big.NewInt(int64(i)))
	}

	for i := 0; i < len(k.ints); i++ {
		grabWord()
		grabInt()
	}
	grabWord()

	// Prealloc and use FillBytes to pad leading zeros.
	k.raw = make([]byte, k.c.totBytes)
	b.FillBytes(k.raw)

}

func (k *HESP) checkTokens(v []string) error {

	checkWord := func(s string) error {
		_, found := BIP39Lookup[s]
		if !found {
			return HESPError(fmt.Sprintf("word not in dictionary: '%s'", s))
		}
		return nil
	}

	checkNum := func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil {
			return HESPError(fmt.Sprintf("token is not a number: %s", s))
		}
		if n < 0 || n >= k.c.intModulus {
			return HESPError(fmt.Sprintf("number out of range: %d", n))
		}
		return nil
	}

	for i, tok := range v {
		var err error
		if i%2 == 0 {
			err = checkWord(tok)
		} else {
			err = checkNum(tok)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *HESP) FromTokens(v []string) error {

	// first check tokens so that tokens not in dictionary
	// or bad numbers will be shown as errors first.
	err := k.checkTokens(v)
	if err != nil {
		return err
	}

	k.ints = make([]int, 0, k.c.numInts)
	k.words = make([]int, 0, k.c.numWords)

	shift := func() string {
		ret := v[0]
		v = v[1:]
		return ret
	}

	grabWord := func() error {
		s := shift()
		s = strings.ToLower(s)
		v, found := BIP39Lookup[s]
		if !found {
			return HESPError(fmt.Sprintf("unknown word: '%s'", s))
		}
		k.words = append(k.words, v)
		return nil
	}

	grabNum := func() error {
		s := shift()
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		if n < 0 || n >= k.c.intModulus {
			return HESPError(fmt.Sprintf("number out of range: %d", n))
		}
		k.ints = append(k.ints, n)
		return nil
	}

	if len(v) != k.c.numInts+k.c.numWords {
		return HESPError(fmt.Sprintf("wrong number of tokens: %d", len(v)))
	}

	for i := 0; i < k.c.numInts; i++ {
		err := grabWord()
		if err != nil {
			return err
		}
		err = grabNum()
		if err != nil {
			return err
		}
	}
	err = grabWord()
	if err != nil {
		return err
	}

	k.fillRaw()
	return nil
}

func HESPToKexSecret(words proto.KexHESP, x *proto.KexSecret) error {
	hesp := NewHESP(KexSeedHESPConfig)
	err := hesp.FromTokens(words)
	if err != nil {
		return err
	}
	if len(hesp.raw) != len(x) {
		return HESPError("wrong length")
	}
	copy((*x)[:], hesp.raw)
	return nil
}

func KexSecretToHESP(x proto.KexSecret) (proto.KexHESP, error) {
	hesp := NewHESP(KexSeedHESPConfig)
	err := hesp.Import(x[:])
	if err != nil {
		return nil, err
	}
	return hesp.Export(), nil
}
