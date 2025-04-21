// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopoSort(t *testing.T) {

	//
	// Imagine this setup:
	//
	//   A < B < C < D < E
	//   A < F < D < E
	//   B < G < E
	//   H < E
	//   I < J
	//

	in := func(c, s string) bool { return strings.Contains(s, c) }

	items := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	items = Reverse(items)
	less := func(a, b string) bool {
		switch a {
		case "A":
			return in(b, "BCDEFG")
		case "B":
			return in(b, "CDEG")
		case "C":
			return in(b, "DE")
		case "D":
			return in(b, "E")
		case "F":
			return in(b, "DE")
		case "G":
			return in(b, "E")
		case "H":
			return in(b, "E")
		case "I":
			return b == "J"
		default:
			return false
		}
	}

	err := TopoSort(items, less)
	require.NoError(t, err)
	idx := func(c string) int {
		for i, v := range items {
			if v == c {
				return i
			}
		}
		t.Fatalf("could not find %s in %v", c, items)
		return 0
	}

	check := make(map[string]bool)
	for _, v := range items {
		if check[v] {
			t.Fatalf("duplicate %s", v)
		}
		check[v] = true
	}
	require.Equal(t, len(check), len(items))

	require.Less(t, idx("A"), idx("B"))
	require.Less(t, idx("B"), idx("C"))
	require.Less(t, idx("C"), idx("D"))
	require.Less(t, idx("D"), idx("E"))
	require.Less(t, idx("A"), idx("F"))
	require.Less(t, idx("F"), idx("D"))
	require.Less(t, idx("B"), idx("G"))
	require.Less(t, idx("G"), idx("E"))
	require.Less(t, idx("H"), idx("E"))
	require.Less(t, idx("I"), idx("J"))

	cycleLess := func(a, b string) bool {
		switch a {
		case "A":
			return b == "B"
		case "B":
			return b == "C"
		case "C":
			return b == "A"
		default:
			return false
		}
	}

	err = TopoSort(items, cycleLess)
	require.Error(t, err)
	require.Equal(t, CycleError{}, err)

}

func TestTopoTraverseOrder(t *testing.T) {

	// Imagine this graph:
	//
	// A -> {B,C,D}
	// B -> {F,M}
	// M -> {E}
	// C -> {F}
	// F -> {E}
	// D -> {G}
	// H -> {D,G}
	// I -> {J}
	// J -> {L}
	// L -> {E}
	//
	// Many orderings work, let's make sure our sort outputs one.

	graph := map[string][]string{
		"A": {"B", "C", "D"},
		"B": {"F", "M"},
		"M": {"E"},
		"C": {"F"},
		"F": {"E"},
		"D": {"G"},
		"H": {"D", "G"},
		"I": {"J"},
		"J": {"K"},
		"L": {"E"},
	}

	res, err := TopoTraverseOrder(graph)
	require.NoError(t, err)

	require.Equal(t, int('M'-'A'+1), len(res))

	idx := func(c string) int {
		for i, v := range res {
			if v == c {
				return i
			}
		}
		t.Fatalf("could not find %s in %v", c, res)
		return 0
	}

	check := make(map[string]bool)
	for _, v := range res {
		if check[v] {
			t.Fatalf("duplicate %s", v)
		}
		check[v] = true
	}
	require.Equal(t, len(check), len(res))

	for k, v := range graph {
		for _, o := range v {
			require.Less(t, idx(k), idx(o))
		}
	}

	veq := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i, v := range a {
			if v != b[i] {
				return false
			}
		}
		return true
	}

	// Assert the result is random. We might get a collision every
	// now and again but we shouldn't get 10 in a row, that would
	// likely mean it's a bug and we're not randomizing our choice.
	var found bool
	for i := 0; !found && i < 10; i++ {
		res2, err := TopoTraverseOrder(graph)
		require.NoError(t, err)
		if !veq(res, res2) {
			found = true
		}
	}
	require.True(t, found)

	cycle := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	}
	res, err = TopoTraverseOrder(cycle)
	require.Error(t, err)
	require.Equal(t, CycleError{}, err)
	require.Nil(t, res)

}
