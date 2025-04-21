// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"context"
	"crypto/x509"
	"os"
	"strings"
)

func GetX509Path(ctx context.Context, s string) (Path, error) {
	if strings.HasPrefix(s, "---") {
		return "", ConfigError("Expected a file path, got a certificate")
	}
	return Path(s), nil
}

func ExpandX509File(ctx context.Context, s string) (string, error) {
	if strings.HasPrefix(s, "---") {
		return s, nil
	}
	ret, err := os.ReadFile(s)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func ExpandCertPool(ctx context.Context, s string) (*x509.CertPool, error) {
	dat, err := ExpandX509File(ctx, s)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(dat)) {
		return nil, TLSError("failed to append cert")
	}
	return pool, nil
}
