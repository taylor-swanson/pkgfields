// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"pkgfields/internal/field"

	"github.com/Masterminds/semver/v3"
)

type Package struct {
	Manifest      Manifest
	BuildManifest *BuildManifest
	Input         *DataStream
	DataStreams   map[string]*DataStream

	srcDir string
}

func (p *Package) Path() string { return p.srcDir }

type Manifest struct {
	Name          string          `yaml:"name"`
	Title         string          `yaml:"title"`
	Version       *semver.Version `yaml:"version"`
	FormatVersion *semver.Version `yaml:"format_version"`
	Type          Type            `yaml:"type"`

	Conditions struct {
		Kibana struct {
			version *semver.Constraints `yaml:"version"`
		} `yaml:"kibana"`
	} `yaml:"conditions"`

	Owner struct {
		Github string `yaml:"github"`
		Type   string `yaml:"type"`
	} `yaml:"owner"`

	srcFile string
}

func (m *Manifest) Path() string { return m.srcFile }

// BuildManifest is the package build manifest.
type BuildManifest struct {
	Dependencies struct {
		ECS struct {
			Reference string `yaml:"reference"`
		} `yaml:"ecs"`
	} `yaml:"dependencies"`

	srcFile string
}

func (m *BuildManifest) Path() string { return m.srcFile }

// DataStreamManifest is the data stream manifest file.
type DataStreamManifest struct {
	Title string `yaml:"title"`
	Type  string `yaml:"type"`

	srcFile string
}

func (m *DataStreamManifest) Path() string { return m.srcFile }

// DataStream is a data stream within the package.
type DataStream struct {
	Manifest DataStreamManifest
	Fields   []field.Field

	srcDir string
}

func (d *DataStream) Path() string { return d.srcDir }
