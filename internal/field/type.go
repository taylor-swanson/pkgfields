// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package field

import (
	"fmt"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeGroup
	TypeAggregateMetricDouble
	TypeAlias
	TypeHistogram
	TypeConstantKeyword
	TypeText
	TypeMatchOnlyText
	TypeKeyword
	TypeLong
	TypeInteger
	TypeShort
	TypeByte
	TypeDouble
	TypeFloat
	TypeHalfFloat
	TypeScaledFloat
	TypeDate
	TypeDateNanos
	TypeBoolean
	TypeBinary
	TypeIntegerRange
	TypeFloatRange
	TypeLongRange
	TypeDoubleRange
	TypeDateRange
	TypeIPRange
	TypeGeoPoint
	TypeObject
	TypeIP
	TypeNested
	TypeFlattened
	TypeWildcard
	TypeVersion
	TypeUnsignedLong
	TypeCountedKeyword
	TypeSemanticText
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
	case TypeGroup:
		return "group"
	case TypeAggregateMetricDouble:
		return "aggregate_metric_double"
	case TypeAlias:
		return "alias"
	case TypeHistogram:
		return "histogram"
	case TypeConstantKeyword:
		return "constant_keyword"
	case TypeText:
		return "text"
	case TypeMatchOnlyText:
		return "match_only_text"
	case TypeKeyword:
		return "keyword"
	case TypeLong:
		return "long"
	case TypeInteger:
		return "integer"
	case TypeShort:
		return "short"
	case TypeByte:
		return "byte"
	case TypeDouble:
		return "double"
	case TypeFloat:
		return "float"
	case TypeHalfFloat:
		return "half_float"
	case TypeScaledFloat:
		return "scaled_float"
	case TypeDate:
		return "date"
	case TypeDateNanos:
		return "date_nanos"
	case TypeBoolean:
		return "boolean"
	case TypeBinary:
		return "binary"
	case TypeIntegerRange:
		return "integer_range"
	case TypeFloatRange:
		return "float_range"
	case TypeLongRange:
		return "long_range"
	case TypeDoubleRange:
		return "double_range"
	case TypeDateRange:
		return "date_range"
	case TypeIPRange:
		return "ip_range"
	case TypeGeoPoint:
		return "geo_point"
	case TypeObject:
		return "object"
	case TypeIP:
		return "ip"
	case TypeNested:
		return "nested"
	case TypeFlattened:
		return "flattened"
	case TypeWildcard:
		return "wildcard"
	case TypeVersion:
		return "version"
	case TypeUnsignedLong:
		return "unsigned_long"
	case TypeCountedKeyword:
		return "counted_keyword"
	case TypeSemanticText:
		return "semantic_text"
	default:
		return "unknown"
	}
}

func ParseType(s string) (Type, error) {
	switch s {
	case "group":
		return TypeGroup, nil
	case "aggregate_metric_double":
		return TypeAggregateMetricDouble, nil
	case "alias":
		return TypeAlias, nil
	case "histogram":
		return TypeHistogram, nil
	case "constant_keyword":
		return TypeConstantKeyword, nil
	case "text":
		return TypeText, nil
	case "match_only_text":
		return TypeMatchOnlyText, nil
	case "keyword":
		return TypeKeyword, nil
	case "long", "number":
		return TypeLong, nil
	case "integer":
		return TypeInteger, nil
	case "short":
		return TypeShort, nil
	case "byte":
		return TypeByte, nil
	case "double":
		return TypeDouble, nil
	case "float":
		return TypeFloat, nil
	case "half_float":
		return TypeHalfFloat, nil
	case "scaled_float":
		return TypeScaledFloat, nil
	case "date":
		return TypeDate, nil
	case "date_nanos":
		return TypeDateNanos, nil
	case "boolean":
		return TypeBoolean, nil
	case "binary":
		return TypeBinary, nil
	case "integer_range":
		return TypeIntegerRange, nil
	case "float_range":
		return TypeFloatRange, nil
	case "long_range":
		return TypeLongRange, nil
	case "double_range":
		return TypeDoubleRange, nil
	case "date_range":
		return TypeDateRange, nil
	case "ip_range":
		return TypeIPRange, nil
	case "geo_point":
		return TypeGeoPoint, nil
	case "object":
		return TypeObject, nil
	case "ip":
		return TypeIP, nil
	case "nested":
		return TypeNested, nil
	case "flattened":
		return TypeFlattened, nil
	case "wildcard":
		return TypeWildcard, nil
	case "version":
		return TypeVersion, nil
	case "unsigned_long":
		return TypeUnsignedLong, nil
	case "counted_keyword":
		return TypeCountedKeyword, nil
	case "semantic_text":
		return TypeSemanticText, nil
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
