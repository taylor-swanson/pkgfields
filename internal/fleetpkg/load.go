// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pkgfields/internal/field"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

type fieldDef struct {
	Name        string     `yaml:"name"`
	External    string     `yaml:"external"`
	Type        string     `yaml:"type"`
	Fields      []fieldDef `yaml:"fields"`
	MultiFields []fieldDef `yaml:"multi_fields"`
}

func loadDataStreamFields(ds *DataStream) error {
	var flatten func(def *fieldDef, parent string)
	flatten = func(def *fieldDef, parent string) {
		// TODO: Should the .* be removed here? Or added as a separate field?
		name := strings.TrimSuffix(field.Join(parent, def.Name), ".*")

		if def.External == "ecs" {
			ds.Fields = append(ds.Fields, field.Field{
				Name: name,
				Kind: field.KindECS,
				Type: field.TypeUnknown, // Type will be expanded later.
			})

			for _, v := range def.Fields {
				flatten(&v, def.Name)
			}
			for _, v := range def.MultiFields {
				flatten(&v, def.Name)
			}

			return
		}

		fieldType := field.TypeKeyword
		if def.Type != "" {
			// TODO: notify of type error?
			if ft, err := field.ParseType(def.Type); err == nil {
				fieldType = ft
			}
		}

		ds.Fields = append(ds.Fields, field.Field{
			Name: name,
			Kind: field.KindVendor,
			Type: fieldType,
		})

		for _, v := range def.Fields {
			flatten(&v, name)
		}
		for _, v := range def.MultiFields {
			flatten(&v, name)
		}
	}

	fieldFiles, err := filepath.Glob(filepath.Join(ds.srcDir, "fields/*.yml"))
	if err != nil {
		return err
	}

	for _, f := range fieldFiles {
		var fieldDefs []fieldDef

		if err = parseYAMLFile(f, &fieldDefs); err != nil {
			return err
		}

		for _, v := range fieldDefs {
			flatten(&v, "")
		}
	}

	slices.SortFunc(ds.Fields, func(a, b field.Field) int {
		if n := cmp.Compare(a.Type, b.Type); n != 0 {
			return n
		}

		return cmp.Compare(a.Name, b.Name)
	})

	return nil
}

func Load(pkgDir string) (*Package, error) {
	var err error

	pkg := Package{srcDir: pkgDir}

	// -------------------------------------------------------------------------
	// Package manifest

	manifestFilename := filepath.Join(pkgDir, "manifest.yml")
	if err = parseYAMLFile(manifestFilename, &pkg.Manifest); err != nil {
		return nil, err
	}
	pkg.Manifest.srcFile = manifestFilename

	// -------------------------------------------------------------------------
	// Build manifest

	BuildManifestFilename := filepath.Join(pkgDir, "_dev", "build", "build.yml")
	var buildManifest BuildManifest
	if err = parseYAMLFile(BuildManifestFilename, &buildManifest); err != nil {
		slog.Warn("Package does not contain a build manifest", slog.String("package", pkg.Manifest.Name), slog.String("filename", BuildManifestFilename))
	} else {
		buildManifest.srcFile = BuildManifestFilename
		pkg.BuildManifest = &buildManifest
	}

	// -------------------------------------------------------------------------
	// Data Streams

	var dataStreamManifests []string
	if pkg.Manifest.Type == TypeInput {
		dataStreamManifests = []string{filepath.Join(pkgDir, "manifest.yml")}
	} else {
		var err error
		pkg.DataStreams = map[string]*DataStream{}
		if dataStreamManifests, err = filepath.Glob(filepath.Join(pkgDir, "data_stream/*/manifest.yml")); err != nil {
			return nil, err
		}
	}

	for _, manifestPath := range dataStreamManifests {
		ds := &DataStream{
			srcDir: filepath.Dir(manifestPath),
		}
		if pkg.Manifest.Type == TypeInput {
			pkg.Input = ds
			if err = loadDataStreamFields(ds); err != nil {
				return nil, err
			}
			break
		} else {
			pkg.DataStreams[filepath.Base(ds.srcDir)] = ds

			if err = parseYAMLFile(manifestPath, &ds.Manifest); err != nil {
				return nil, err
			}
			if err = loadDataStreamFields(ds); err != nil {
				return nil, err
			}
		}
	}

	return &pkg, nil
}

func parseYAMLFile(filename string, v any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading YAML file %s: %w", filename, err)
	}
	if err = yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("error parsing YAML file %s: %w", filename, err)
	}

	return nil
}
