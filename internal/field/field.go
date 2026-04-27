// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package field

type Field struct {
	Name string
	Kind Kind
	Type Type
}

// Join joins a field name to a parent dotted notation path.
func Join(parent, name string) string {
	if parent == "" {
		return name
	}

	return parent + "." + name
}
