package golang

import (
	"bufio"
	"bytes"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
	"strings"
	"text/template"
)

type gqlTmplCtx struct {
	ModelPackage string
	Enums        []Enum
	Structs      []Struct
	GoQueries    []Query
	SqlcVersion  string

	// TODO: Race conditions
	SourceName string

	OmitSqlcVersion bool
}

func generateGql(
	req *plugin.GenerateRequest,
	options *opts.Options,
	enums []Enum,
	structs []Struct,
	queries []Query,
) (*plugin.GenerateResponse, error) {
	i := &importer{
		Options: options,
		Queries: queries,
		Enums:   enums,
		Structs: structs,
	}

	tctx := gqlTmplCtx{
		ModelPackage:    options.GqlModelPackage,
		Enums:           enums,
		Structs:         structs,
		SqlcVersion:     req.SqlcVersion,
		OmitSqlcVersion: options.OmitSqlcVersion,
	}

	funcMap := template.FuncMap{
		"lowerTitle": sdk.LowerTitle,
		"comment":    sdk.DoubleSlashComment,
		"escape":     sdk.EscapeBacktick,
		"imports":    i.Imports,
		"hasImports": i.HasImports,
		"hasPrefix":  strings.HasPrefix,
	}

	tmpl := template.Must(
		template.New("table").
			Funcs(funcMap).
			ParseFS(
				templates,
				"templates/graphql/*.tmpl",
			),
	)

	output := map[string]string{}

	execute := func(name, templateName string) error {
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		tctx.SourceName = name
		err := tmpl.ExecuteTemplate(w, templateName, &tctx)
		w.Flush()
		if err != nil {
			return err
		}

		if !strings.HasSuffix(name, ".graphql") {
			name += ".graphql"
		}
		output[name] = string(b.Bytes())
		return nil
	}

	gqlFileName := "schema.graphql"

	if err := execute(gqlFileName, "modelsGqlFile"); err != nil {
		return nil, err
	}

	if options.GqlGenCommonParts {
		if err := execute("common.graphql", "commonGqlFile"); err != nil {
			return nil, err
		}
	}
	resp := plugin.GenerateResponse{}

	for filename, code := range output {
		if options.GqlOut != "" {
			filename = options.GqlOut + "/" + filename
		}
		resp.Files = append(
			resp.Files, &plugin.File{
				Name:     filename,
				Contents: []byte(code),
			},
		)
	}

	return &resp, nil
}
