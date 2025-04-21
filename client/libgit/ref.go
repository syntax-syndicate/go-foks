// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libgit

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type ReferenceStorage struct {
	adptr *adapter
}

func (r *ReferenceStorage) SetReference(ref *plumbing.Reference) error {
	return r.adptr.putReference(context.Background(), ref)
}

// CheckAndSetReference sets the reference `new`, but if `old` is
// not `nil`, it first checks that the current stored value for
// `old.Name()` matches the given reference value in `old`.  If
// not, it returns an error and doesn't update `new`.
func (r *ReferenceStorage) CheckAndSetReference(new, old *plumbing.Reference) error {
	return r.adptr.putReferenceConditional(context.Background(), new, old)
}
func (r *ReferenceStorage) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	return r.adptr.getReference(context.Background(), n)
}
func (r *ReferenceStorage) IterReferences() (storer.ReferenceIter, error) {
	return r.adptr.openReferenceIter(context.Background())
}

func (r *ReferenceStorage) RemoveReference(n plumbing.ReferenceName) error {
	return r.adptr.unlinkReference(context.Background(), n)
}

// CountLooseRefs is unemplemnted for now, we'll implement it later
// for packing references.
func (r *ReferenceStorage) CountLooseRefs() (int, error) {
	return 0, nil
}

// PackRefs is unemplemnted for now, we'll implement it later
// for packing references.
func (r *ReferenceStorage) PackRefs() error {
	return nil
}

var _ storer.ReferenceStorer = (*ReferenceStorage)(nil)
