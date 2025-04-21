// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package common

import (
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type MockVanityHelper struct {
	ProbeCA X509CA
}

func NewMockVanityHelper(pca X509CA) *MockVanityHelper {
	return &MockVanityHelper{ProbeCA: pca}
}

func (h *MockVanityHelper) CheckCNAMEResolvesTo(
	m shared.MetaContext,
	from proto.Hostname,
	to proto.Hostname,
) error {
	res := m.G().CnameResolver().ResolveSingle(from)
	if res == "" {
		return core.NotFoundError("CNAME resolution")
	}
	res = proto.Hostname(res.String() + ".")
	if !res.WithTrailingDot().NormEq(to.WithTrailingDot()) {
		return core.HostMismatchError{}
	}
	return nil
}

func (h *MockVanityHelper) SetCNAME(
	m shared.MetaContext,
	from proto.Hostname,
	to proto.Hostname,
) error {
	m.G().CnameResolver().Add(from, to)
	return nil
}

func (h *MockVanityHelper) ClearCNAME(
	m shared.MetaContext,
	hn proto.Hostname,
) error {
	m.G().CnameResolver().Remove(hn)
	return nil
}

func (h *MockVanityHelper) Autocert(
	m shared.MetaContext,
	pkg shared.AutocertPackage,
) error {
	res := m.G().CnameResolver().ResolveSingle(pkg.Hostname)
	if res == "" {
		return core.NotFoundError("CNAME resolution")
	}
	return shared.DoAutocertViaClient(m, time.Minute, pkg)
}

var _ shared.VanityHelper = (*MockVanityHelper)(nil)
