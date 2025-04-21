// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto/rand"
	"math/big"
)

// RamdomInt uses crypto/rand to pick a number in the range [0, n).
// Will return an error if reading from rand.Reader failed.
func RandomInt(n int) (int, error) {
	// Convert n to *big.Int for the crypto/rand.Int function
	max := big.NewInt(int64(n))
	randomNumber, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	// Convert the *big.Int to int
	return int(randomNumber.Int64()), nil
}

// TopoSort is a topological sort of the vector v, which supports a
// partial-ordering defined by the less function. Will return nil
// if there was a cycle.
func TopoSort[
	S ~[]E,
	E any,
](v S, less func(E, E) bool) error {

	type node struct {
		e   E
		idx int
		out []int
		in  map[int]bool
	}

	nodes := make([]node, 0, len(v))

	for i, x := range v {
		nodes = append(nodes, node{x, i, nil, make(map[int]bool)})
	}

	for i := range nodes {
		// We need an actual referece to the slot here, since we'll
		// be mutating it.
		x := &nodes[i]
		for j, y := range nodes {
			if i != j && less(x.e, y.e) {
				x.out = append(x.out, j)
				y.in[i] = true
			}
		}
	}

	var noIns []*node
	retPtr := 0

	// Start nodes all have no incoming edges
	for i := range nodes {
		n := &nodes[i]
		if len(n.in) == 0 {
			noIns = append(noIns, n)
		}
	}

	// Kahn's Algorithm: https://en.wikipedia.org/wiki/Topological_sorting
	for len(noIns) > 0 {
		curr := noIns[0]
		noIns = noIns[1:]
		v[retPtr] = curr.e
		retPtr++
		for _, out := range curr.out {
			nxt := &nodes[out]
			delete(nxt.in, curr.idx)
			if len(nxt.in) == 0 {
				noIns = append(noIns, nxt)
			}
		}
	}

	// If any nodes in the graph still have incoming edges, there was a cycle
	for _, n := range nodes {
		if len(n.in) > 0 {
			return CycleError{}
		}
	}

	return nil
}

// TopoTraverseOrder takes a graph represented as a map of nodes to their
// outgoing edges, and returns a topological ordering of the nodes. Assumes
// no cycles are present in the graph, but will return an error if one is found.
func TopoTraverseOrder[K comparable](m map[K][]K) ([]K, error) {

	type node struct {
		k   K
		out []K
		in  map[K]bool
	}

	nodes := make(map[K]*node)

	// make a node for each of our input nodes that has outgoing edges
	for k, out := range m {
		nodes[k] = &node{k, out, make(map[K]bool)}
	}

	// we'll still be left with some nodes that don't have outgoing edges,
	// so be sure to add them with a second loop through.
	for _, out := range m {
		for _, o := range out {
			if _, ok := nodes[o]; !ok {
				nodes[o] = &node{o, nil, make(map[K]bool)}
			}
		}
	}

	// collect incoming edges
	for k, n := range nodes {
		for _, o := range n.out {
			if _, ok := nodes[o]; ok {
				nodes[o].in[k] = true
			}
		}
	}

	var noIns []*node
	ret := make([]K, 0, len(nodes))

	for _, n := range nodes {
		if len(n.in) == 0 {
			noIns = append(noIns, n)
		}
	}

	// Randomly pick an item from the list, remove it from the list.
	// We'd like the sort order to be random so that 2 similar users
	// will key in different orders, i.e., won't race.
	rsel := func(v []*node) ([]*node, *node, error) {
		if len(v) == 0 {
			return nil, nil, nil
		}
		idx, err := RandomInt(len(v))
		if err != nil {
			return nil, nil, err
		}
		ret := v[idx]
		v[idx] = v[0]
		return v[1:], ret, nil
	}

	// Kahn's Algorithm: https://en.wikipedia.org/wiki/Topological_sorting
	for len(noIns) > 0 {
		nextNoIns, curr, err := rsel(noIns)
		if err != nil {
			return nil, err
		}
		ret = append(ret, curr.k)
		noIns = nextNoIns
		for _, out := range curr.out {
			nxt, ok := nodes[out]
			if !ok {
				continue
			}
			delete(nxt.in, curr.k)
			if len(nxt.in) == 0 {
				noIns = append(noIns, nxt)
			}
		}
	}

	for _, n := range nodes {
		if len(n.in) > 0 {
			return nil, CycleError{}
		}
	}

	return ret, nil
}
