# sqlc-gen-go

See [Building from source](#building-from-source) and [Migrating from sqlc's built-in Go codegen](#migrating-from-sqlcs-built-in-go-codegen) if you want to use a modified fork in your project.

## Fork purpose
This is a fork of the original sqlc-gen-go plugin that adds the ability to change the default schema.
Imagine that you have a database with multiple schemas one per your module in code. 
Each module has its own set of entity names. 

For example, you have a module `auth` 
that has entities like AccessToken, RefreshToken, etc. 
All these entities are stored in the `auth` schema in the database.

By default, sqlc-gen-go generates code for this module with the prefix Auth, for example, AuthAccessToken, AuthRefreshToken, etc.

If you want to delete a prefix, you can use the `default_schema` option in the plugin configuration.

```yaml
version: '2'
plugins:
  - name: golang
    wasm:
      url: "https://github.com/debugger84/sqlc-gen-go/releases/download/v1.3.1/sqlc-gen-go.wasm"
      sha256: "fe6e5a2b75153ecba02b0c30bf4a11db2120bef537b650299473da133d272bf4"
sql:
  - schema:
      - "migration"
      - "../types/migration"
    queries: "queries"
    engine: "postgresql"
    codegen:
      - plugin: golang
        out: "./"
        options:
          package: "storage"
          sql_package: "pgx/v4"
          out: "./"
          default_schema: "auth"
```

In this case, the current fork of the plugin will generate code without the Auth prefix.


The second feature is the ability to debug the SQL code generation process.
If you want to debug the plugin follow the next steps:
1. Use the `process` option in the plugin configuration instead of wasm. 
Also, you need to send the `DUMP_REQUEST_FILE` environment variable, adding the env parameter to yaml. 
```yaml
version: '2'
plugins:
  - name: golang
    process:
      cmd: "../sqlc-gen-go/bin/sqlc-gen-go"
    env:
      - DUMP_REQUEST_FILE
```
2. Run the sqlc command with the `DUMP_REQUEST_FILE` environment variable.
```bash
DUMP_REQUEST_FILE=/tmp/1234 sqlc generate
```
3. The plugin will generate the request file in the specified directory.
4. Configure your Goland IDE to run the plugin project with the env variable. Fill the field in Run\Debug Configurations->your configuration->Environment field with the value RESTORE_REQUEST_FILE=/tmp/1234
5. Put a breakpoint everywhere you want, run the project in Debug mode, and enjoy the debugging process.

## Usage

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: https://downloads.sqlc.dev/plugin/sqlc-gen-go_1.3.0.wasm
    sha256: e8206081686f95b461daf91a307e108a761526c6768d6f3eca9781b0726b7ec8
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      sql_package: pgx/v5
```

## Building from source

Assuming you have the Go toolchain set up, from the project root you can simply `make all`.

```sh
make all
```

This will produce a standalone binary and a WASM blob in the `bin` directory.
They don't depend on each other, they're just two different plugin styles. You can
use either with sqlc, but we recommend WASM and all of the configuration examples
here assume you're using a WASM plugin.

To use a local WASM build with sqlc, just update your configuration with a `file://`
URL pointing at the WASM blob in your `bin` directory:

```yaml
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
```

As-of sqlc v1.24.0 the `sha256` is optional, but without it sqlc won't cache your
module internally which will impact performance.

## Migrating from sqlc's built-in Go codegen

We’ve worked hard to make switching to sqlc-gen-go as seamless as possible. Let’s say you’re generating Go code today using a sqlc.yaml configuration that looks something like this:

```yaml
version: 2
sql:
- schema: "query.sql"
  queries: "query.sql"
  engine: "postgresql"
  gen:
    go:
      package: "db"
      out: "db"
      emit_json_tags: true
      emit_pointers_for_null_types: true
      query_parameter_limit: 5
      overrides:
      - column: "authors.id"
        go_type: "your/package.SomeType"
      rename:
        foo: "bar"
```

To use the sqlc-gen-go WASM plugin for Go codegen, your config will instead look something like this:

```yaml
version: 2
plugins:
- name: golang
  wasm:
    url: https://downloads.sqlc.dev/plugin/sqlc-gen-go_1.3.0.wasm
    sha256: e8206081686f95b461daf91a307e108a761526c6768d6f3eca9781b0726b7ec8
sql:
- schema: "query.sql"
  queries: "query.sql"
  engine: "postgresql"
  codegen:
  - plugin: golang
    out: "db"
    options:
      package: "db"
      emit_json_tags: true
      emit_pointers_for_null_types: true
      query_parameter_limit: 5
      overrides:
      - column: "authors.id"
        go_type: "your/package.SomeType"
      rename:
        foo: "bar"
```

The differences are:
* An additional top-level `plugins` list with an entry for the Go codegen WASM plugin. If you’ve built the plugin from source you’ll want to use a `file://` URL. The `sha256` field is required, but will be optional in the upcoming sqlc v1.24.0 release.
* Within the `sql` block, rather than `gen` with `go` nested beneath you’ll have a `codegen` list with an entry referencing the plugin name from the top-level `plugins` list. All options from the current `go` configuration block move as-is into the `options` block within `codegen`. The only special case is `out`, which moves up a level into the `codegen` configuration itself.

### Global overrides and renames

If you have global overrides or renames configured, you’ll need to move those to the new top-level `options` field. Replace the existing `go` field name with the name you gave your plugin in the `plugins` list. We’ve used `"golang"` in this example.

If your existing configuration looks like this:

```yaml
version: "2"
overrides:
  go:
    rename:
      id: "Identifier"
    overrides:
    - db_type: "timestamptz"
      nullable: true
      engine: "postgresql"
      go_type:
        import: "gopkg.in/guregu/null.v4"
        package: "null"
        type: "Time"
...
```

Then your updated configuration would look something like this:

```yaml
version: "2"
plugins:
- name: golang
  wasm:
    url: https://downloads.sqlc.dev/plugin/sqlc-gen-go_1.3.0.wasm
    sha256: e8206081686f95b461daf91a307e108a761526c6768d6f3eca9781b0726b7ec8
options:
  golang:
    rename:
      id: "Identifier"
    overrides:
    - db_type: "timestamptz"
      nullable: true
      engine: "postgresql"
      go_type:
        import: "gopkg.in/guregu/null.v4"
        package: "null"
        type: "Time"
...
```
