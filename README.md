# pkgfields

`pkgfields` is a command-line tool that extracts field definitions from
[Elastic integration packages](https://github.com/elastic/integrations). It
reads a package directory, flattens the fields declared in each data stream
(or input package), resolves references to [ECS](https://github.com/elastic/ecs)
fields against the package's declared ECS reference, and prints the result as
a table or JSON.

For each field it reports:

- `name` — the fully-qualified field name
- `kind` — `ecs` if the field is sourced from ECS, otherwise `vendor`
- `type` — the data type (from ECS for ECS fields, from the package for vendor fields)

## Build and install

Requires Go 1.26+.

```sh
# Build a local binary
go build -o pkgfields .

# Or install into $GOBIN / $GOPATH/bin
go install .
```

## Usage

```sh
pkgfields [flags] PKG_DIR [PKG_DIR...]
```

`PKG_DIR` is the path to an integration package directory (the directory
containing `manifest.yml`). Multiple directories may be provided.

### Flags

- `-data-streams` — comma-separated list of data streams to filter on
- `-json` — output as JSON
- `-minify` — minify JSON output (only with `-json`)
- `-cache-dir` — directory for cached ECS field definitions (default `.pkgfields-cache`; empty string disables the cache)
- `-debug` — enable debug logging

### Examples

Print fields for a single integration package as a table:

```sh
pkgfields ~/code/elastic/integrations/packages/nginx
```

Restrict output to specific data streams:

```sh
pkgfields -data-streams access,error ~/code/elastic/integrations/packages/nginx
```

Emit JSON for multiple packages:

```sh
pkgfields -json \
  ~/code/elastic/integrations/packages/nginx \
  ~/code/elastic/integrations/packages/apache
```

Emit JSON for all packages:

```sh
pkgfields -json ~/code/elastic/integrations/packages/*
```

ECS field definitions are fetched from
`https://raw.githubusercontent.com/elastic/ecs/<ref>/generated/ecs/ecs_flat.yml`
based on the ECS reference declared in the package's `_dev/build/build.yml`,
and cached locally under `-cache-dir`.
