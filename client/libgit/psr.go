// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libgit

import (
	"context"
	"io"
	"time"

	remhelp "github.com/foks-proj/go-git-remhelp"
)

type PackSyncRemote struct {
	adptr *adapter
}

func newPackSyncRemote(
	a *adapter,
) *PackSyncRemote {
	return &PackSyncRemote{
		adptr: a,
	}
}

func (p *PackSyncRemote) FetchNewIndices(
	ctx context.Context,
	since time.Time,
) ([]remhelp.RawIndex, error) {
	return p.adptr.fetchNewIndices(ctx, since)
}

func (p *PackSyncRemote) FetchPackData(
	ctx context.Context,
	name remhelp.IndexName,
	wc io.Writer,
) error {
	return p.adptr.fetchPackData(ctx, name, wc)
}

func (p *PackSyncRemote) PushPackData(
	ctx context.Context,
	name remhelp.IndexName,
	rc io.Reader,
) error {
	return p.adptr.pushPackData(ctx, name, rc)
}

func (p *PackSyncRemote) PushPackIndex(
	ctx context.Context,
	name remhelp.IndexName,
	rc io.Reader,
) error {
	return p.adptr.pushPackIndex(ctx, name, rc)
}

func (p *PackSyncRemote) HasIndex(
	ctx context.Context,
	name remhelp.IndexName,
) (bool, error) {
	return p.adptr.hasIndex(ctx, name)
}

var _ remhelp.PackSyncRemoter = (*PackSyncRemote)(nil)
