// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

type EmptyConfigError struct{}

func (e EmptyConfigError) Error() string {
	return "no config loaded; but it was needed"
}

func NewEmptyConfigError() error {
	return EmptyConfigError{}
}
