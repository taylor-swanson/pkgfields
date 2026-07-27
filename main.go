// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/andrewkroh/go-package-spec/pkgreader"
	"github.com/andrewkroh/go-package-spec/pkgspec"
	"gopkg.in/yaml.v3"

	"github.com/taylor-swanson/pkgfields/internal/version"
)

func joinFieldName(parent, name string) string {
	if parent == "" {
		return name
	}

	return parent + "." + name
}

func splitFieldName(name string) (string, bool) {
	i := strings.LastIndex(name, ".")
	if i == -1 {
		return name, false
	}

	return name[:i], true
}

type Doc map[string]any

func (d Doc) ExtractFields(result map[string]struct{}, parent string) {
	for k, v := range d {
		name := joinFieldName(parent, k)

		switch t := v.(type) {
		case map[string]any:
			(Doc(t)).ExtractFields(result, name)
		case []any:
			for _, sliceValue := range t {
				if elem, ok := sliceValue.(map[string]any); ok {
					(Doc(elem)).ExtractFields(result, name)
				}
			}
		default:
			result[name] = struct{}{}
		}
	}
}

type pipelineExpected struct {
	Expected []Doc
}

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
	pkgDirs           []string
	cacheDir          string
	debug             bool
	minified          bool
	outputCSV         bool
	outputJSON        bool
	showVersion       bool
)

func usage() {
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "pkgfields [flags] PKG_DIR [PKG_DIR...]")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Elastic fleet package field extractor.")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Args:")
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "  PKG_DIR\n\tpath to package directory, multiple directories may be provided")
	_, _ = fmt.Fprintln(flag.CommandLine.Output())
	_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
	flag.PrintDefaults()
}

func parseArgs() {
	flag.Usage = usage
	flag.StringVar(&cacheDir, "cache-dir", ".pkgfields-cache", "directory to store cached files (use empty string to disable cache)")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&outputCSV, "csv", false, "output as CSV")
	flag.BoolVar(&outputJSON, "json", false, "output as JSON")
	flag.BoolVar(&minified, "minify", false, "minify output JSON")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Var(&filterDataStreams, "data-streams", "filter on a comma-separated list of data streams")

	flag.Parse()

	pkgDirs = flag.Args()
}

func fetchRemoteFile(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote file: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

type fieldInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Type string `json:"type,omitempty"`
}

type ecsFieldDef struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Normalize   any    `yaml:"normalize"`
}

type ecsResolver struct {
	fields map[string]*pkgspec.ECSFieldDefinition
}

func (r *ecsResolver) Lookup(name string) *pkgspec.ECSFieldDefinition {
	return r.fields[name]
}

func newECSResolver(ref string) (*ecsResolver, error) {
	isArray := func(v any) bool {
		switch t := v.(type) {
		case string:
			if t == "array" {
				return true
			}
		case []any:
			for _, tv := range t {
				switch vv := tv.(type) {
				case string:
					if vv == "array" {
						return true
					}
				}
			}
		}

		return false
	}

	ref = strings.TrimPrefix(ref, "git@")

	cacheFile := filepath.Join(cacheDir, "ecs-"+ref+".json")

	if cacheDir != "" {
		if _, err := os.Stat(cacheFile); err == nil {
			var r ecsResolver
			raw, err := os.ReadFile(cacheFile)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(raw, &r.fields); err != nil {
				return nil, err
			}

			return &r, nil
		}
	}

	raw, err := fetchRemoteFile("https://raw.githubusercontent.com/elastic/ecs/" + ref + "/generated/ecs/ecs_flat.yml")
	if err != nil {
		return nil, err
	}
	var ecsFieldDefs map[string]ecsFieldDef
	if err = yaml.Unmarshal(raw, &ecsFieldDefs); err != nil {
		return nil, err
	}

	r := ecsResolver{
		fields: make(map[string]*pkgspec.ECSFieldDefinition, len(ecsFieldDefs)),
	}
	for k, v := range ecsFieldDefs {
		r.fields[k] = &pkgspec.ECSFieldDefinition{
			DataType:    v.Type,
			Description: v.Description,
			Array:       isArray(v.Normalize),
		}
		if v.Type == "geo_point" {
			r.fields[k+".lat"] = &pkgspec.ECSFieldDefinition{
				DataType: v.Type,
			}
			r.fields[k+".lon"] = &pkgspec.ECSFieldDefinition{
				DataType: v.Type,
			}
			r.fields[k+".type"] = &pkgspec.ECSFieldDefinition{
				DataType: v.Type,
			}
			r.fields[k+".coordinates"] = &pkgspec.ECSFieldDefinition{
				DataType: v.Type,
				Array:    true,
			}
		}
	}

	if cacheDir != "" {
		if err = os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, err
		}
		data, err := json.Marshal(r.fields)
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(cacheFile, data, 0o644); err != nil {
			return nil, err
		}
	}

	return &r, nil
}

func processFieldsFiles(files map[string]*pkgreader.FieldsFile, resolver *ecsResolver, ds, pkg string) map[string]fieldInfo {
	fields := map[string]fieldInfo{}

	for _, file := range files {
		var ecsFunc func(name string) *pkgspec.ECSFieldDefinition
		if resolver != nil {
			ecsFunc = func(name string) *pkgspec.ECSFieldDefinition {
				return resolver.Lookup(name)
			}
		}

		flattened := pkgspec.FlattenFields(file.Fields, ecsFunc)

		for _, f := range flattened {
			info := fieldInfo{
				Name: strings.TrimSuffix(f.Name, ".*"),
			}

			if f.External == pkgspec.FieldExternalECS {
				info.Kind = "ecs"
				if f.ECS != nil {
					info.Type = f.ECS.DataType
				}
			} else {
				if strings.HasPrefix(info.Name, "ocsf.") {
					info.Kind = "ocsf"
				} else {
					info.Kind = "vendor"
				}
				info.Type = string(f.Type)

				if info.Type == "geo_point" {
					fields[info.Name+".lat"] = fieldInfo{
						Name: info.Name + ".lat",
						Kind: info.Kind,
						Type: info.Type,
					}
					fields[info.Name+".lon"] = fieldInfo{
						Name: info.Name + ".lon",
						Kind: info.Kind,
						Type: info.Type,
					}
					fields[info.Name+".type"] = fieldInfo{
						Name: info.Name + ".type",
						Kind: info.Kind,
						Type: info.Type,
					}
					fields[info.Name+".coordinates"] = fieldInfo{
						Name: info.Name + ".coordinates",
						Kind: info.Kind,
						Type: info.Type,
					}
				}
			}

			if info.Type == "" {
				slog.Warn("Field has no type", slog.String("field", info.Name), slog.String("kind", info.Kind), slog.String("data_stream", ds), slog.String("package", pkg))
			}

			fields[info.Name] = info
		}
	}

	return fields
}

func processDataStreamTests(testDir string, fields map[string]fieldInfo, resolver *ecsResolver) error {
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(testDir, "*-expected.json"))
	if err != nil {
		return err
	}

	extracted := map[string]struct{}{}

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		var expected pipelineExpected
		if err = json.Unmarshal(raw, &expected); err != nil {
			return err
		}

		for _, v := range expected.Expected {
			v.ExtractFields(extracted, "")
		}
	}

	for k := range extracted {
		if _, ok := fields[k]; ok {
			continue
		}

		ecsDef := resolver.Lookup(k)
		if ecsDef != nil {
			fields[k] = fieldInfo{
				Name: k,
				Kind: "ecs",
				Type: ecsDef.DataType,
			}
			continue
		}

		if k == "o365.audit.Actor" {
			slog.Error("")
		}

		found := false
		split, didSplit := splitFieldName(k)
		for didSplit {
			if f, ok := fields[split]; ok {
				if f.Type == "nested" || f.Type == "object" || f.Type == "flattened" {
					found = true
				}
				break
			}

			split, didSplit = splitFieldName(split)
		}

		if !found {
			slog.Warn("Failed to resolve field", slog.String("name", k))
		}
	}

	return nil
}

func doExtract() error {
	type pkgResult struct {
		Package      string                          `json:"package"`
		Version      string                          `json:"version"`
		ECSReference string                          `json:"ecs_reference"`
		Input        map[string]fieldInfo            `json:"input,omitempty"`
		DataStreams  map[string]map[string]fieldInfo `json:"data_streams,omitempty"`
	}

	var results []pkgResult
	for _, pkgDir := range pkgDirs {
		pkg, err := pkgreader.Read(pkgDir)
		if err != nil {
			return err
		}

		if strings.Contains(strings.ToLower(pkg.Manifest().Title), "deprecated") {
			slog.Debug("Skipping deprecated package", slog.String("package", pkg.Manifest().Name))
			continue
		}

		result := pkgResult{
			Package: pkg.Manifest().Name,
			Version: pkg.Manifest().Version,
		}
		var resolver *ecsResolver
		// Workaround for pkgspec not reading build manifest on non-integration type packages.
		if pkg.Build == nil {
			var buildManifest pkgspec.BuildManifest
			if data, err := os.ReadFile(filepath.Join(pkgDir, "_dev", "build", "build.yml")); err == nil {
				if err = yaml.Unmarshal(data, &buildManifest); err == nil {
					pkg.Build = &buildManifest
				}
			}
		}
		if pkg.Build != nil {
			result.ECSReference = pkg.Build.Dependencies.ECS.Reference
			if resolver, err = newECSResolver(result.ECSReference); err != nil {
				slog.Error("Failed to create ECS resolver", slog.String("error", err.Error()))
			}
		}

		if pkg.Manifest().Type == pkgspec.ManifestTypeInput {
			result.Input = processFieldsFiles(pkg.Fields, resolver, "", pkg.Manifest().Name)
		} else {
			result.DataStreams = make(map[string]map[string]fieldInfo)
			for name, ds := range pkg.DataStreams {
				if len(filterDataStreams) > 0 && !slices.Contains(filterDataStreams, name) {
					continue
				}
				referenceFields := processFieldsFiles(ds.Fields, resolver, name, pkg.Manifest().Name)

				if err = processDataStreamTests(filepath.Join(pkgDir, ds.Path(), "_dev", "test", "pipeline"), referenceFields, resolver); err != nil {
					return err
				}

				result.DataStreams[name] = referenceFields
			}
		}

		results = append(results, result)
	}

	var err error
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		if !minified {
			enc.SetIndent("", "  ")
		}
		if err = enc.Encode(results); err != nil {
			return err
		}
	} else if outputCSV {
		fmt.Println("package,version,field,kind,type")
		for _, result := range results {
			if len(result.Input) > 0 {
				printFieldsCSV(os.Stdout, result.Package, result.Version, result.Input)
			} else {
				names := make([]string, 0, len(result.DataStreams))
				for k := range result.DataStreams {
					names = append(names, k)
				}
				sort.Strings(names)

				for _, name := range names {
					printFieldsCSV(os.Stdout, result.Package+"."+name, result.Version, result.DataStreams[name])
				}
			}
		}
	} else {
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		for _, result := range results {
			fmt.Println("Package:", result.Package)
			fmt.Println("Version:", result.Version)
			if result.ECSReference != "" {
				fmt.Println("ECS Reference:", result.ECSReference)
			}
			fmt.Println()

			if len(result.Input) > 0 {
				_, _ = fmt.Fprintln(tw, "Name\tKind\tType")
				_, _ = fmt.Fprintln(tw, "----\t----\t----")
				printFieldsText(tw, result.Input)
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
					printFieldsText(tw, result.DataStreams[name])
					_ = tw.Flush()
					fmt.Println("")
				}
			}
		}
	}

	return nil
}

func printFieldsText(w io.Writer, fieldMap map[string]fieldInfo) {
	fields := make([]fieldInfo, 0, len(fieldMap))
	for _, v := range fieldMap {
		fields = append(fields, v)
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Kind < fields[j].Kind
	})

	for _, v := range fields {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.Name, v.Kind, v.Type)
	}
}

func printFieldsCSV(w io.Writer, pkg, version string, fieldMap map[string]fieldInfo) {
	fields := make([]fieldInfo, 0, len(fieldMap))
	for _, v := range fieldMap {
		fields = append(fields, v)
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Kind < fields[j].Kind
	})

	for _, v := range fields {
		_, _ = fmt.Fprintf(w, "%s,%s,%s,%s,%s\n", pkg, version, v.Name, v.Kind, v.Type)
	}
}

func main() {
	parseArgs()

	if showVersion {
		fmt.Printf("%s version %s [commit %v]\n", version.Name, version.Version, version.Commit)
		os.Exit(0)
	}
	if len(pkgDirs) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := doExtract(); err != nil {
		slog.Error("Error running app", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
