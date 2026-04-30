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
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/andrewkroh/go-package-spec/pkgreader"
	"github.com/andrewkroh/go-package-spec/pkgspec"
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

type fieldInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Type string `json:"type,omitempty"`
}

func processFieldsFiles(files map[string]*pkgreader.FieldsFile) []fieldInfo {
	var fields []fieldInfo

	var flatten func(f pkgspec.Field, parent string)
	flatten = func(f pkgspec.Field, parent string) {
		name := f.Name
		if parent != "" {
			name = parent + "." + name
		}

		if f.Type != pkgspec.FieldTypeGroup {
			fi := fieldInfo{
				Name: name,
			}
			if f.External == pkgspec.FieldExternalECS {
				fi.Kind = "ecs"
			} else {
				fi.Kind = "vendor"
				fi.Type = string(f.Type)
			}
			fields = append(fields, fi)
		}

		for _, v := range f.Fields {
			flatten(v, name)
		}
	}

	for _, file := range files {
		for _, f := range file.Fields {
			flatten(f, "")
		}
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Kind < fields[j].Kind
	})

	return fields
}

func doExtract() error {
	type pkgResult struct {
		Package      string                 `json:"package"`
		Version      string                 `json:"version"`
		ECSReference string                 `json:"ecs_reference"`
		Input        []fieldInfo            `json:"input,omitempty"`
		DataStreams  map[string][]fieldInfo `json:"data_streams,omitempty"`
	}

	pkg, err := pkgreader.Read(pkgDir)
	if err != nil {
		return err
	}

	result := pkgResult{
		Package: pkg.Manifest().Name,
		Version: pkg.Manifest().Version,
	}
	if pkg.Build != nil {
		result.ECSReference = pkg.Build.Dependencies.ECS.Reference
	}

	if pkg.Manifest().Type == pkgspec.ManifestTypeInput {
		result.Input = processFieldsFiles(pkg.Fields)
	} else {
		result.DataStreams = make(map[string][]fieldInfo)
		for name, ds := range pkg.DataStreams {
			if len(filterDataStreams) > 0 && !slices.Contains(filterDataStreams, name) {
				continue
			}
			result.DataStreams[name] = processFieldsFiles(ds.Fields)
		}
	}

	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		if err = enc.Encode(result); err != nil {
			return err
		}
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		fmt.Println("Package:", result.Package)
		fmt.Println("Version:", result.Version)
		if result.ECSReference != "" {
			fmt.Println("ECS Reference:", result.ECSReference)
		}
		fmt.Println()

		if len(result.Input) > 0 {
			_, _ = fmt.Fprintln(tw, "Name\tKind\tType")
			_, _ = fmt.Fprintln(tw, "----\t----\t----")
			for _, f := range result.Input {
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Name, f.Kind, f.Type)
			}
			_ = tw.Flush()
		} else {
			names := make([]string, 0, len(result.DataStreams))
			for k := range result.DataStreams {
				names = append(names, k)
			}
			sort.Strings(names)

			for _, name := range names {
				fmt.Println("Data stream:", name)
				fmt.Println("")
				_, _ = fmt.Fprintln(tw, "Name\tKind\tType")
				_, _ = fmt.Fprintln(tw, "----\t----\t----")
				for _, f := range result.DataStreams[name] {
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Name, f.Kind, f.Type)
				}
				_ = tw.Flush()
				fmt.Println("")
			}
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
