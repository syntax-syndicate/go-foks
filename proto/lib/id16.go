// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/json"
	"fmt"
)

type ID16er interface {
	Type() ID16Type
	ToID16() *ID16
	Bytes() []byte    // Satisfied by the RPC compiler, but it's immutable
	BytesMut() []byte // Mutable buffer
	Name() string
	StringErr() (string, error)
}

func (i ID16) Type() ID16Type {
	return ID16Type(i[0])
}

var _ ID16er = (*VHostID)(nil)
var _ ID16er = (*PlanID)(nil)
var _ ID16er = (*PriceID)(nil)
var _ ID16er = (*CancelID)(nil)
var _ ID16er = (*TeamRSVPLocal)(nil)
var _ ID16er = (*TeamRSVPRemote)(nil)
var _ ID16er = (*LocalInstanceID)(nil)
var _ ID16er = (*PermissionToken)(nil)
var _ ID16er = (*ReservationToken)(nil)
var _ ID16er = (*AutocertID)(nil)
var _ ID16er = (*OAuth2SessionID)(nil)

func (v *VHostID) Type() ID16Type              { return ID16Type_VHost }
func (v *VHostID) ToID16() *ID16               { return (*ID16)(v) }
func (v *VHostID) BytesMut() []byte            { return v[:] }
func (v *VHostID) Name() string                { return "vhost" }
func (v *VHostID) ImportFromDB(b []byte) error { return id16ImportFromBytes(v, b) }
func (v *VHostID) StringErr() (string, error)  { return v.ToID16().StringErr() }
func (v *VHostID) String() string              { return stringEatErr(v) }
func (v VHostID) ExportToDB() []byte           { return v.Bytes() }
func (v *VHostID) DOMString() string           { return v.String()[1:] }
func (v *VHostID) IsZero() bool                { return IsZero(v[:]) }

func (p *PlanID) Type() ID16Type              { return ID16Type_Plan }
func (p *PlanID) ToID16() *ID16               { return (*ID16)(p) }
func (p *PlanID) BytesMut() []byte            { return p[:] }
func (p *PlanID) Name() string                { return "plan" }
func (p *PlanID) ImportFromDB(b []byte) error { return id16ImportFromBytes(p, b) }
func (p *PlanID) StringErr() (string, error)  { return p.ToID16().StringErr() }
func (p *PlanID) String() string              { return stringEatErr(p) }
func (p *PlanID) IsZero() bool                { return IsZero(p[:]) }
func (p *PlanID) ExportToDB() []byte          { return p.Bytes() }
func (a *PlanID) Eq(b PlanID) bool            { return id16eq(a, &b) }

func (p *PriceID) Type() ID16Type              { return ID16Type_Price }
func (p *PriceID) ToID16() *ID16               { return (*ID16)(p) }
func (p *PriceID) BytesMut() []byte            { return p[:] }
func (p *PriceID) Name() string                { return "price" }
func (p *PriceID) ImportFromDB(b []byte) error { return id16ImportFromBytes(p, b) }
func (p *PriceID) StringErr() (string, error)  { return p.ToID16().StringErr() }
func (p *PriceID) ExportToDB() []byte          { return p.Bytes() }
func (p *PriceID) IsZero() bool                { return IsZero(p[:]) }
func (a *PriceID) Eq(b PriceID) bool           { return id16eq(a, &b) }
func (a *PriceID) String() string              { return stringEatErr(a) }

func (c *CancelID) Type() ID16Type              { return ID16Type_Cancel }
func (c *CancelID) ToID16() *ID16               { return (*ID16)(c) }
func (c *CancelID) BytesMut() []byte            { return c[:] }
func (c *CancelID) Name() string                { return "cancel" }
func (c *CancelID) ImportFromDB(b []byte) error { return id16ImportFromBytes(c, b) }
func (c *CancelID) StringErr() (string, error)  { return c.ToID16().StringErr() }
func (c *CancelID) ExportToDB() []byte          { return c.Bytes() }
func (c *CancelID) IsZero() bool                { return IsZero(c[:]) }
func (a *CancelID) Eq(b CancelID) bool          { return id16eq(a, &b) }
func (a *CancelID) String() string              { return stringEatErr(a) }

func (t *TeamRSVPLocal) Type() ID16Type              { return ID16Type_TeamRSVPLocal }
func (t *TeamRSVPLocal) ToID16() *ID16               { return (*ID16)(t) }
func (t *TeamRSVPLocal) BytesMut() []byte            { return t[:] }
func (t *TeamRSVPLocal) Name() string                { return "teamRSVPLocal" }
func (t *TeamRSVPLocal) ImportFromDB(b []byte) error { return id16ImportFromBytes(t, b) }
func (t *TeamRSVPLocal) StringErr() (string, error)  { return t.ToID16().StringErr() }
func (t *TeamRSVPLocal) ExportToDB() []byte          { return t.Bytes() }
func (t *TeamRSVPLocal) IsZero() bool                { return IsZero(t[:]) }
func (a *TeamRSVPLocal) Eq(b TeamRSVPLocal) bool     { return id16eq(a, &b) }
func (a *TeamRSVPLocal) String() string              { return stringEatErr(a) }

func (t *TeamRSVPRemote) Type() ID16Type              { return ID16Type_TeamRSVPRemote }
func (t *TeamRSVPRemote) ToID16() *ID16               { return (*ID16)(t) }
func (t *TeamRSVPRemote) BytesMut() []byte            { return t[:] }
func (t *TeamRSVPRemote) Name() string                { return "teamRSVPRemote" }
func (t *TeamRSVPRemote) ImportFromDB(b []byte) error { return id16ImportFromBytes(t, b) }
func (t *TeamRSVPRemote) StringErr() (string, error)  { return t.ToID16().StringErr() }
func (t *TeamRSVPRemote) ExportToDB() []byte          { return t.Bytes() }
func (t *TeamRSVPRemote) IsZero() bool                { return IsZero(t[:]) }
func (a *TeamRSVPRemote) Eq(b TeamRSVPRemote) bool    { return id16eq(a, &b) }
func (a *TeamRSVPRemote) String() string              { return stringEatErr(a) }

func (t *LocalInstanceID) Type() ID16Type               { return ID16Type_LocalInstance }
func (t *LocalInstanceID) ToID16() *ID16                { return (*ID16)(t) }
func (t *LocalInstanceID) BytesMut() []byte             { return t[:] }
func (t *LocalInstanceID) Name() string                 { return "localInstance" }
func (t *LocalInstanceID) ImportFromDB(b []byte) error  { return id16ImportFromBytes(t, b) }
func (t *LocalInstanceID) StringErr() (string, error)   { return t.ToID16().StringErr() }
func (t *LocalInstanceID) ExportToDB() []byte           { return t.Bytes() }
func (t *LocalInstanceID) IsZero() bool                 { return IsZero(t[:]) }
func (a *LocalInstanceID) Eq(b LocalInstanceID) bool    { return id16eq(a, &b) }
func (a *LocalInstanceID) String() string               { return stringEatErr(a) }
func (a LocalInstanceID) MarshalJSON() ([]byte, error)  { return id16MarshalJSON(&a) }
func (a *LocalInstanceID) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (p *PermissionToken) Type() ID16Type               { return ID16Type_PermissionToken }
func (p *PermissionToken) ToID16() *ID16                { return (*ID16)(p) }
func (p *PermissionToken) BytesMut() []byte             { return p[:] }
func (p *PermissionToken) Name() string                 { return "permissionToken" }
func (p *PermissionToken) ImportFromDB(b []byte) error  { return id16ImportFromBytes(p, b) }
func (p *PermissionToken) StringErr() (string, error)   { return p.ToID16().StringErr() }
func (p *PermissionToken) ExportToDB() []byte           { return p.Bytes() }
func (p *PermissionToken) IsZero() bool                 { return IsZero(p[:]) }
func (a *PermissionToken) Eq(b PermissionToken) bool    { return id16eq(a, &b) }
func (a *PermissionToken) String() string               { return stringEatErr(a) }
func (a PermissionToken) MarshalJSON() ([]byte, error)  { return id16MarshalJSON(&a) }
func (a *PermissionToken) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (r *ReservationToken) Type() ID16Type               { return ID16Type_ReservationToken }
func (r *ReservationToken) ToID16() *ID16                { return (*ID16)(r) }
func (r *ReservationToken) BytesMut() []byte             { return r[:] }
func (r *ReservationToken) Name() string                 { return "reservationToken" }
func (r *ReservationToken) ImportFromDB(b []byte) error  { return id16ImportFromBytes(r, b) }
func (r *ReservationToken) StringErr() (string, error)   { return r.ToID16().StringErr() }
func (r *ReservationToken) ExportToDB() []byte           { return r.Bytes() }
func (r *ReservationToken) IsZero() bool                 { return IsZero(r[:]) }
func (a *ReservationToken) Eq(b ReservationToken) bool   { return id16eq(a, &b) }
func (a *ReservationToken) String() string               { return stringEatErr(a) }
func (a ReservationToken) MarshalJSON() ([]byte, error)  { return id16MarshalJSON(&a) }
func (a *ReservationToken) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (a *AutocertID) Type() ID16Type               { return ID16Type_Autocert }
func (a *AutocertID) ToID16() *ID16                { return (*ID16)(a) }
func (a *AutocertID) BytesMut() []byte             { return a[:] }
func (a *AutocertID) Name() string                 { return "AutocertID" }
func (a *AutocertID) ImportFromDB(b []byte) error  { return id16ImportFromBytes(a, b) }
func (a *AutocertID) StringErr() (string, error)   { return a.ToID16().StringErr() }
func (a *AutocertID) ExportToDB() []byte           { return a.Bytes() }
func (a *AutocertID) IsZero() bool                 { return IsZero(a[:]) }
func (a *AutocertID) Eq(b ReservationToken) bool   { return id16eq(a, &b) }
func (a *AutocertID) String() string               { return stringEatErr(a) }
func (a AutocertID) MarshalJSON() ([]byte, error)  { return id16MarshalJSON(&a) }
func (a *AutocertID) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (a *OAuth2SessionID) Type() ID16Type               { return ID16Type_OAuth2Session }
func (a *OAuth2SessionID) ToID16() *ID16                { return (*ID16)(a) }
func (a *OAuth2SessionID) BytesMut() []byte             { return a[:] }
func (a *OAuth2SessionID) Name() string                 { return "OAuth2SessionID" }
func (a *OAuth2SessionID) ImportFromDB(b []byte) error  { return id16ImportFromBytes(a, b) }
func (a *OAuth2SessionID) StringErr() (string, error)   { return a.ToID16().StringErr() }
func (a *OAuth2SessionID) ExportToDB() []byte           { return a.Bytes() }
func (a *OAuth2SessionID) IsZero() bool                 { return IsZero(a[:]) }
func (a *OAuth2SessionID) Eq(b OAuth2SessionID) bool    { return id16eq(a, &b) }
func (a *OAuth2SessionID) String() string               { return stringEatErr(a) }
func (a *OAuth2SessionID) MarshalJSON() ([]byte, error) { return id16MarshalJSON(a) }
func (a *OAuth2SessionID) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (a *SSOConfigID) Type() ID16Type               { return ID16Type_SSOConfig }
func (a *SSOConfigID) ToID16() *ID16                { return (*ID16)(a) }
func (a *SSOConfigID) BytesMut() []byte             { return a[:] }
func (a *SSOConfigID) Name() string                 { return "SSOConfigID" }
func (a *SSOConfigID) ImportFromDB(b []byte) error  { return id16ImportFromBytes(a, b) }
func (a *SSOConfigID) StringErr() (string, error)   { return a.ToID16().StringErr() }
func (a *SSOConfigID) ExportToDB() []byte           { return a.Bytes() }
func (a *SSOConfigID) IsZero() bool                 { return IsZero(a[:]) }
func (a *SSOConfigID) Eq(b SSOConfigID) bool        { return id16eq(a, &b) }
func (a *SSOConfigID) String() string               { return stringEatErr(a) }
func (a *SSOConfigID) MarshalJSON() ([]byte, error) { return id16MarshalJSON(a) }
func (a *SSOConfigID) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (a *CKSKeyID) Type() ID16Type               { return ID16Type_CKSKey }
func (a *CKSKeyID) ToID16() *ID16                { return (*ID16)(a) }
func (a *CKSKeyID) BytesMut() []byte             { return a[:] }
func (a *CKSKeyID) Name() string                 { return "DBEncKey" }
func (a *CKSKeyID) ImportFromDB(b []byte) error  { return id16ImportFromBytes(a, b) }
func (a *CKSKeyID) StringErr() (string, error)   { return a.ToID16().StringErr() }
func (a *CKSKeyID) ExportToDB() []byte           { return a.Bytes() }
func (a *CKSKeyID) IsZero() bool                 { return IsZero(a[:]) }
func (a *CKSKeyID) Eq(b SSOConfigID) bool        { return id16eq(a, &b) }
func (a *CKSKeyID) String() string               { return stringEatErr(a) }
func (a *CKSKeyID) MarshalJSON() ([]byte, error) { return id16MarshalJSON(a) }
func (a *CKSKeyID) UnmarshalJSON(b []byte) error { return id16UnmarshalJSON(b, a) }

func (i *ID16) ToPlanID() (*PlanID, error)                 { return id16ToSubclass[PlanID](i) }
func (i *ID16) ToVHostID() (*VHostID, error)               { return id16ToSubclass[VHostID](i) }
func (i *ID16) ToPriceID() (*PriceID, error)               { return id16ToSubclass[PriceID](i) }
func (i *ID16) ToCancelID() (*CancelID, error)             { return id16ToSubclass[CancelID](i) }
func (i *ID16) ToAutocertID() (*AutocertID, error)         { return id16ToSubclass[AutocertID](i) }
func (i *ID16) ToTeamRSVPLocal() (*TeamRSVPLocal, error)   { return id16ToSubclass[TeamRSVPLocal](i) }
func (i *ID16) ToTeamRSVPRemote() (*TeamRSVPRemote, error) { return id16ToSubclass[TeamRSVPRemote](i) }
func (i *ID16) ToCKSKeyID() (*CKSKeyID, error)             { return id16ToSubclass[CKSKeyID](i) }
func (i *ID16) ToOAuth2SessionID() (*OAuth2SessionID, error) {
	return id16ToSubclass[OAuth2SessionID](i)
}
func (i *ID16) ToReservationToken() (*ReservationToken, error) {
	return id16ToSubclass[ReservationToken](i)
}
func (i *ID16) ToLocalInstanceID() (*LocalInstanceID, error) {
	return id16ToSubclass[LocalInstanceID](i)
}
func (i *ID16) ToPermissionToken() (*PermissionToken, error) {
	return id16ToSubclass[PermissionToken](i)
}
func (i *ID16) ToOAuth2ConfigID() (*SSOConfigID, error) {
	return id16ToSubclass[SSOConfigID](i)
}

func (t TeamRSVP) MarshalJSON() ([]byte, error) { return id16MarshalJSON(t) }

func NewCancelID() (*CancelID, error) { return RandomID16er[CancelID]() }
func NewVHostID() (*VHostID, error)   { return RandomID16er[VHostID]() }
func NilCancelID() []byte             { return []byte{0x00} }

func id16eq(a, b ID16er) bool {
	return hmac.Equal(a.Bytes(), b.Bytes())
}

func id16ImportFromBytes(out ID16er, in []byte) error {
	outBytes := out.BytesMut()
	if len(outBytes) != len(in) {
		return DataError("bad " + out.Name() + " id; wrong length")
	}
	copy(outBytes, in)
	if ID16Type(outBytes[0]) != out.Type() {
		return DataError(
			fmt.Sprintf("wrong type for ID16; wanted %s, got %d",
				out.Name(), outBytes[0]),
		)
	}
	return nil
}

func stringEatErr(i interface{ StringErr() (string, error) }) string {
	ret, err := i.StringErr()
	if err != nil {
		ret = "error"
	}
	return ret
}

func (t ID16Type) RandomID() (*ID16, error) {
	var buf [16]byte

	n, err := rand.Read(buf[:])
	if err != nil {
		return nil, err
	}
	if n != len(buf) {
		return nil, DataError("short random ready for new ID16")
	}
	return t.MakeID16(buf[:])
}

func RandomID16er[
	T any,
	PT interface {
		*T
		ID16er
	},
]() (*T, error) {
	var ret T
	var pt PT = &ret
	byt := pt.BytesMut()
	_, err := rand.Read(byt[1:])
	if err != nil {
		return nil, err
	}
	byt[0] = byte(pt.Type())
	return &ret, nil
}

func id16ToSubclass[
	T any,
	PT interface {
		*T
		ID16er
	},
](i *ID16) (*T, error) {
	var ret T
	var pt PT = &ret
	if err := id16ImportFromBytes(pt, i[:]); err != nil {
		return nil, err
	}
	return &ret, nil
}

func (t ID16Type) IsValid() bool {
	switch t {
	case ID16Type_Plan, ID16Type_Cancel, ID16Type_Price,
		ID16Type_VHost, ID16Type_TeamRSVPLocal, ID16Type_TeamRSVPRemote,
		ID16Type_LocalInstance, ID16Type_PermissionToken, ID16Type_ReservationToken,
		ID16Type_Autocert, ID16Type_OAuth2Session, ID16Type_SSOConfig, ID16Type_CKSKey:
		return true
	default:
		return false
	}
}

func (t ID16Type) HasLeadingDot() bool {
	switch t {
	case ID16Type_TeamRSVPLocal, ID16Type_TeamRSVPRemote, ID16Type_OAuth2Session:
		return false
	default:
		return true
	}
}

func (t ID16Type) MakeID16(buf []byte) (*ID16, error) {
	var ret ID16
	ret[0] = byte(t)
	if len(ret[1:]) != len(buf) {
		return nil, DataError("wrong id16 length")
	}
	copy(ret[1:], buf)
	return &ret, nil
}

func (i ID16String) Parse() (*ID16, error) {
	if len(i) < 10 {
		return nil, DataError("ID16 was too short")
	}
	hasLeadingDot := false
	if i[0] == '.' {
		hasLeadingDot = true
		i = i[1:]
	}
	b0, err := B62DecodeByte(i[0])
	if err != nil {
		return nil, err
	}
	typ := ID16Type(b0)
	if !typ.IsValid() {
		return nil, DataError("invalid id16 type")
	}
	if typ.HasLeadingDot() && !hasLeadingDot {
		return nil, DataError("missing leading dot")
	}
	dat, err := B62Decode(string(i[1:]))
	if err != nil {
		return nil, err
	}
	return typ.MakeID16(dat)
}

func (i *ID16) Data() []byte {
	return (*i)[1:]
}

func (i ID16) ID16StringErr() (ID16String, error) {
	ret, err := i.StringErr()
	if err != nil {
		return "", err
	}
	return ID16String(ret), nil
}

func (i ID16String) String() string {
	return string(i)
}

func (i *ID16) StringErr() (string, error) {
	return PrefixedB62EncodeDotOpt(i.Type().HasLeadingDot(), byte(i.Type()), i.Data())
}

func NewTeamRSVPWithLocal(t TeamRSVPLocal) TeamRSVP {
	return TeamRSVP(t)
}

func NewTeamRSVPWithRemote(t TeamRSVPRemote) TeamRSVP {
	return TeamRSVP(t)
}

func (t TeamRSVP) Remote() (*TeamRSVPRemote, error) {
	tmp := ID16(t)
	return tmp.ToTeamRSVPRemote()
}

func (t TeamRSVP) Local() (*TeamRSVPLocal, error) {
	tmp := ID16(t)
	return tmp.ToTeamRSVPLocal()
}

func (t *TeamRSVP) Sel() (*TeamRSVPLocal, *TeamRSVPRemote, error) {
	tmp := ID16(*t)
	typ := tmp.Type()
	switch typ {
	case ID16Type_TeamRSVPLocal:
		l, err := tmp.ToTeamRSVPLocal()
		return l, nil, err
	case ID16Type_TeamRSVPRemote:
		r, err := tmp.ToTeamRSVPRemote()
		return nil, r, err
	default:
		return nil, nil, DataError("bad team join req token")
	}
}

func (t TeamRSVP) String() TeamRSVPString {
	tmp := ID16(t)
	s, err := tmp.StringErr()
	if err != nil {
		s = "error"
	}
	return TeamRSVPString(s)
}

func (t TeamRSVP) StringErr() (string, error) {
	tmp := ID16(t)
	return tmp.StringErr()
}

func (t TeamRSVPString) Parse() (*TeamRSVP, error) {
	tmp := ID16String(string(t))
	idp, err := tmp.Parse()
	if err != nil {
		return nil, err
	}
	typ := idp.Type()
	switch typ {
	case ID16Type_TeamRSVPLocal, ID16Type_TeamRSVPRemote:
		return (*TeamRSVP)(idp), nil
	default:
		return nil, DataError("bad team join req token")
	}
}

func (o OAuth2SessionIDString) Parse() (*OAuth2SessionID, error) {
	tmp := ID16String(string(o))
	idp, err := tmp.Parse()
	if err != nil {
		return nil, err
	}
	typ := idp.Type()
	switch typ {
	case ID16Type_OAuth2Session:
		return (*OAuth2SessionID)(idp), nil
	default:
		return nil, DataError("bad OAuth2 session ID")
	}
}

func id16MarshalJSON(i StringErrer) ([]byte, error) {
	s, err := i.StringErr()
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

func id16UnmarshalJSON(b []byte, i ID16er) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	tmp := ID16String(s)
	id, err := tmp.Parse()
	if err != nil {
		return err
	}
	targ := i.BytesMut()
	if len(id) != len(targ) {
		return DataError("bad id16 length")
	}
	copy(targ[:], id.Bytes())
	return nil
}

func (t *TeamRSVP) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	ts := TeamRSVPString(s)
	tmp, err := ts.Parse()
	if err != nil {
		return err
	}
	copy((*t)[:], (*tmp)[:])
	return nil
}
