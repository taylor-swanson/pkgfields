// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"pkgfields/internal/field"
	"pkgfields/internal/fleetpkg"
)

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *stringSliceFlag) Set(value string) error {
	vals := strings.Split(value, ",")
	for _, v := range vals {
		*f = append(*f, strings.TrimSpace(v))
	}

	return nil
}

var (
	filterDataStreams stringSliceFlag
	pkgDir            string
	debug             bool
	outputJSON        bool
)

func usage() {
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "pkgfields [flags] PKG_DIR")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Elastic fleet package field extractor.")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Args:")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  PKG_DIR\n\tpath to package directory")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
	flag.PrintDefaults()
}

func parseArgs() {
	flag.Usage = usage
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&outputJSON, "json", false, "output as JSON")
	flag.Var(&filterDataStreams, "data-streams", "filter on a comma-separated list of data streams")

	flag.Parse()

	if len(flag.Args()) >= 1 {
		pkgDir = flag.Arg(0)
	}
}

func doExtract() error {
	pkg, err := fleetpkg.Load(pkgDir, filterDataStreams...)
	if err != nil {
		return err
	}

	if outputJSON {
		type outputField struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
			Type string `json:"type,omitempty"`
		}
		type result struct {
			Package      string                   `json:"package"`
			Version      string                   `json:"version"`
			ECSReference string                   `json:"ecs_reference"`
			Input        []outputField            `json:"input,omitempty"`
			DataStreams  map[string][]outputField `json:"data_streams,omitempty"`
		}

		res := result{
			Package: pkg.Manifest.Name,
			Version: pkg.Manifest.Version.String(),
		}
		if pkg.BuildManifest != nil {
			res.ECSReference = pkg.BuildManifest.Dependencies.ECS.Reference
		}
		if pkg.Input != nil {
			res.Input = make([]outputField, 0, len(pkg.Input.Fields))
			for _, f := range pkg.Input.Fields {
				var typeStr string
				if f.Kind != field.KindECS {
					typeStr = f.Type.String()
				}
				res.Input = append(res.Input, outputField{
					Name: f.Name,
					Kind: f.Kind.String(),
					Type: typeStr,
				})
			}
		} else {
			res.DataStreams = make(map[string][]outputField, len(pkg.DataStreams))
			for dsName, ds := range pkg.DataStreams {

				for _, f := range ds.Fields {
					var typeStr string
					if f.Kind != field.KindECS {
						typeStr = f.Type.String()
					}
					res.DataStreams[dsName] = append(res.DataStreams[dsName], outputField{
						Name: f.Name,
						Kind: f.Kind.String(),
						Type: typeStr,
					})
				}
			}
		}

		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(res)
	}

	fmt.Println("Package:", pkg.Manifest.Name)
	if pkg.BuildManifest != nil {
		fmt.Println("ECS Reference:", pkg.BuildManifest.Dependencies.ECS.Reference)
	}
	fmt.Println()
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if pkg.Input != nil {
		fmt.Println("Input Fields:")
		_, _ = fmt.Fprintln(tw, "Name\tKind\tType\n----\t----\t----")
		for _, f := range pkg.Input.Fields {
			var typeStr string
			if f.Kind != field.KindECS {
				typeStr = f.Type.String()
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Name, f.Kind.String(), typeStr)
		}
		_ = tw.Flush()
	} else {
		dsNames := make([]string, 0, len(pkg.DataStreams))
		for dsName := range pkg.DataStreams {
			dsNames = append(dsNames, dsName)
		}
		sort.Strings(dsNames)

		for _, dsName := range dsNames {
			fmt.Println("Data Stream:", dsName)
			fmt.Println()
			_, _ = fmt.Fprintln(tw, "Name\tKind\tType\n----\t----\t----")
			for _, f := range pkg.DataStreams[dsName].Fields {
				var typeStr string
				if f.Kind != field.KindECS {
					typeStr = f.Type.String()
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Name, f.Kind.String(), typeStr)
			}
			_ = tw.Flush()
			fmt.Println()
		}
	}

	return nil
}

func main() {
	parseArgs()

	if pkgDir == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := doExtract(); err != nil {
		slog.Error("Error running app", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
