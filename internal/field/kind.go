// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package field

import (
	"fmt"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindComputed
	KindVendor
	KindECS
)

func (k *Kind) String() string {
	switch *k {
	case KindComputed:
		return "computed"
	case KindVendor:
		return "vendor"
	case KindECS:
		return "ecs"
	default:
		return "unknown"
	}
}

func (k *Kind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *Kind) UnmarshalText(text []byte) error {
	parsed, err := ParseKind(string(text))
	if err != nil {
		return err
	}

	*k = parsed
	return nil
}

func ParseKind(s string) (Kind, error) {
	switch s {
	case "computed":
		return KindComputed, nil
	case "vendor":
		return KindVendor, nil
	case "ecs":
		return KindECS, nil
	}

	return KindUnknown, fmt.Errorf("unknown kind %q", s)
}
