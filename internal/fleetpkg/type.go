// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"fmt"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeIntegration
	TypeInput
	TypeContent
)

func (t *Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Type) UnmarshalText(text []byte) error {
	parsed, err := ParseType(string(text))
	if err != nil {
		return err
	}

	*t = parsed
	return nil
}

func (t *Type) String() string {
	switch *t {
	case TypeIntegration:
		return "integration"
	case TypeInput:
		return "input"
	case TypeContent:
		return "content"
	default:
		return "unknown"
	}
}

func ParseType(s string) (Type, error) {
	switch s {
	case "integration":
		return TypeContent, nil
	case "input":
		return TypeInput, nil
	case "content":
		return TypeContent, nil
	}

	return TypeUnknown, fmt.Errorf("unknown type %q", s)
}

func MustParseType(s string) Type {
	t, err := ParseType(s)
	if err != nil {
		panic(err)
	}

	return t
}
