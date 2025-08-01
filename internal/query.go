package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

type QueryValue struct {
	Emit        bool
	EmitPointer bool
	Name        string
	DBName      string // The name of the field in the database. Only set if Struct==nil.
	Struct      *Struct
	Typ         string
	SQLDriver   opts.SQLDriver

	// Column is kept so late in the generation process around to differentiate
	// between mysql slices and pg arrays
	Column *plugin.Column
}

func (v QueryValue) EmitStruct() bool {
	return v.Emit
}

func (v QueryValue) IsStruct() bool {
	return v.Struct != nil
}

func (v QueryValue) IsPointer() bool {
	return v.EmitPointer && v.Struct != nil
}

func (v QueryValue) isEmpty() bool {
	return v.Typ == "" && v.Name == "" && v.Struct == nil
}

type Argument struct {
	Name string
	Type string
}

func (v QueryValue) Pair() string {
	var out []string
	for _, arg := range v.Pairs() {
		out = append(out, arg.Name+" "+arg.Type)
	}
	return strings.Join(out, ",")
}

// Return the argument name and type for query methods. Should only be used in
// the context of method arguments.
func (v QueryValue) Pairs() []Argument {
	if v.isEmpty() {
		return nil
	}
	if !v.EmitStruct() && v.IsStruct() {
		var out []Argument
		for _, f := range v.Struct.Fields {
			out = append(
				out, Argument{
					Name: escape(toLowerCase(f.Name)),
					Type: f.Type,
				},
			)
		}
		return out
	}
	return []Argument{
		{
			Name: escape(v.Name),
			Type: v.DefineType(),
		},
	}
}

func (v QueryValue) SlicePair() string {
	if v.isEmpty() {
		return ""
	}
	return v.Name + " []" + v.DefineType()
}

func (v QueryValue) Type() string {
	if v.Typ != "" {
		return v.Typ
	}
	if v.Struct != nil {
		return v.Struct.Name
	}
	panic("no type for QueryValue: " + v.Name)
}

func (v *QueryValue) DefineType() string {
	t := v.Type()
	if v.IsPointer() {
		return "*" + t
	}
	return t
}

func (v *QueryValue) ReturnName() string {
	if v.IsPointer() {
		return "&" + escape(v.Name)
	}
	return escape(v.Name)
}

func (v QueryValue) UniqueFields() []Field {
	seen := map[string]struct{}{}
	fields := make([]Field, 0, len(v.Struct.Fields))

	for _, field := range v.Struct.Fields {
		if _, found := seen[field.Name]; found {
			continue
		}
		seen[field.Name] = struct{}{}
		fields = append(fields, field)
	}

	return fields
}

func (v QueryValue) Params() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if v.Struct == nil {
		if !v.Column.IsSqlcSlice && strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array("+escape(v.Name)+")")
		} else {
			out = append(out, escape(v.Name))
		}
	} else {
		for _, f := range v.Struct.Fields {
			if !f.HasSqlcSlice() && strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array("+escape(v.VariableForField(f))+")")
			} else {
				out = append(out, escape(v.VariableForField(f)))
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

func (v QueryValue) PagedParams() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if v.Struct == nil {
		if !v.Column.IsSqlcSlice && strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array("+escape(v.Name)+")")
		} else {
			out = append(out, escape(v.Name))
		}
	} else {
		for _, f := range v.Struct.Fields {
			if !f.HasSqlcSlice() && strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array("+escape(v.VariableForField(f))+")")
			} else {
				if f.Name == "Limit" {
					out = append(out, escape(v.VariableForField(f))+"+1")
					continue
				}
				out = append(out, escape(v.VariableForField(f)))
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

func (v QueryValue) ParamsTotal() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if v.Struct == nil {
		if !v.Column.IsSqlcSlice && strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array("+escape(v.Name)+")")
		} else {
			out = append(out, escape(v.Name))
		}
	} else {
		for _, f := range v.Struct.Fields {
			if f.Name == "Limit" || f.Name == "Offset" {
				continue
			}
			if !f.HasSqlcSlice() && strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array("+escape(v.VariableForField(f))+")")
			} else {
				out = append(out, escape(v.VariableForField(f)))
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

func (v QueryValue) ColumnNames() []string {
	if v.Struct == nil {
		return []string{v.DBName}
	}
	names := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		names[i] = f.DBName
	}
	return names
}

func (v QueryValue) ColumnNamesAsGoSlice() string {
	if v.Struct == nil {
		return fmt.Sprintf("[]string{%q}", v.DBName)
	}
	escapedNames := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		if f.Column != nil && f.Column.OriginalName != "" {
			escapedNames[i] = fmt.Sprintf("%q", f.Column.OriginalName)
		} else {
			escapedNames[i] = fmt.Sprintf("%q", f.DBName)
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

// When true, we have to build the arguments to q.db.QueryContext in addition to
// munging the SQL
func (v QueryValue) HasSqlcSlices() bool {
	if v.Struct == nil {
		return v.Column != nil && v.Column.IsSqlcSlice
	}
	for _, v := range v.Struct.Fields {
		if v.Column.IsSqlcSlice {
			return true
		}
	}
	return false
}

func (v QueryValue) Scan() string {
	var out []string
	if v.Struct == nil {
		if strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array(&"+v.Name+")")
		} else {
			out = append(out, "&"+v.Name)
		}
	} else {
		for _, f := range v.Struct.Fields {

			// append any embedded fields
			if len(f.EmbedFields) > 0 {
				for _, embed := range f.EmbedFields {
					if strings.HasPrefix(embed.Type, "[]") && embed.Type != "[]byte" && !v.SQLDriver.IsPGX() {
						out = append(out, "pq.Array(&"+v.Name+"."+f.Name+"."+embed.Name+")")
					} else {
						out = append(out, "&"+v.Name+"."+f.Name+"."+embed.Name)
					}
				}
				continue
			}

			if strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array(&"+v.Name+"."+f.Name+")")
			} else {
				out = append(out, "&"+v.Name+"."+f.Name)
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

// Deprecated: This method does not respect the Emit field set on the
// QueryValue. It's used by the go-sql-driver-mysql/copyfromCopy.tmpl and should
// not be used other places.
func (v QueryValue) CopyFromMySQLFields() []Field {
	// fmt.Printf("%#v\n", v)
	if v.Struct != nil {
		return v.Struct.Fields
	}
	return []Field{
		{
			Name:   v.Name,
			DBName: v.DBName,
			Type:   v.Typ,
		},
	}
}

func (v QueryValue) VariableForField(f Field) string {
	if !v.IsStruct() {
		return v.Name
	}
	if !v.EmitStruct() {
		return toLowerCase(f.Name)
	}
	return v.Name + "." + f.Name
}

type CursorField struct {
	Field Field
	IsAsc bool
}

// A struct used to generate methods and fields on the Queries struct
type Query struct {
	Cmd              string
	Comments         []string
	MethodName       string
	FieldName        string
	ConstantName     string
	SQL              string
	SQLPaginated     string
	SourceName       string
	Ret              QueryValue
	Arg              QueryValue
	Paginated        bool
	CursorPagination bool
	CursorFields     []CursorField

	// Used for :copyfrom
	Table *plugin.Identifier
}

func (q Query) hasRetType() bool {
	scanned := q.Cmd == metadata.CmdOne || q.Cmd == metadata.CmdMany ||
		q.Cmd == metadata.CmdBatchMany || q.Cmd == metadata.CmdBatchOne
	return scanned && !q.Ret.isEmpty()
}

func (q Query) TableIdentifierAsGoSlice() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("%q", p))
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

func (q Query) TableIdentifierForMySQL() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("`%s`", p))
		}
	}
	return strings.Join(escapedNames, ".")
}

func (q Query) SQLTotal() string {
	if q.Paginated {
		return fmt.Sprintf("SELECT count(*) as c FROM (%s) as total_res LIMIT 1", q.SQL)
	}
	return ""
}

func (q Query) ReturnedType() string {
	if q.Cmd == metadata.CmdOne {
		return q.Ret.DefineType()
	}
	if q.Cmd != metadata.CmdMany {
		return ""
	}
	if q.Paginated {
		if q.CursorPagination {
			return fmt.Sprintf("%sConnection", q.Ret.DefineType())
		}
		return fmt.Sprintf("%sPage", q.Ret.DefineType())
	}
	return fmt.Sprintf("[]%s", q.Ret.DefineType())
}

func (q Query) QueryParams() string {
	if q.Arg.isEmpty() {
		return ""
	}
	if q.Paginated {
		pageParams := q.Arg.PagedParams()
		if q.CursorPagination {
			pageParamsLines := strings.Split(pageParams, ",")
			pageParamsItems := make([]string, 0, len(pageParamsLines))
			for _, line := range pageParamsLines {
				param := strings.Trim(strings.TrimSpace(line), "\n")
				if param != "" {
					pageParamsItems = append(pageParamsItems, param)
					continue
				}
			}
			for _, field := range q.CursorFields {
				pageParamsItems = append(pageParamsItems, "cursor."+field.Field.Name)
			}
			if len(pageParamsItems) <= 3 {
				return strings.Join(pageParamsItems, ",")
			}
			pageParamsItems = append(pageParamsItems, "")
			return "\n" + strings.Join(pageParamsItems, ",\n")
		}
	}
	return q.Arg.Params()
}
