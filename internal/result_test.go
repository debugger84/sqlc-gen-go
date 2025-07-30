package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestPutOutColumns_ForZeroColumns(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{
			cmd:  metadata.CmdExec,
			want: false,
		},
		{
			cmd:  metadata.CmdExecResult,
			want: false,
		},
		{
			cmd:  metadata.CmdExecRows,
			want: false,
		},
		{
			cmd:  metadata.CmdExecLastId,
			want: false,
		},
		{
			cmd:  metadata.CmdMany,
			want: true,
		},
		{
			cmd:  metadata.CmdOne,
			want: true,
		},
		{
			cmd:  metadata.CmdCopyFrom,
			want: false,
		},
		{
			cmd:  metadata.CmdBatchExec,
			want: false,
		},
		{
			cmd:  metadata.CmdBatchMany,
			want: true,
		},
		{
			cmd:  metadata.CmdBatchOne,
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(
			tc.cmd, func(t *testing.T) {
				query := &plugin.Query{
					Cmd:     tc.cmd,
					Columns: []*plugin.Column{},
				}
				got := putOutColumns(query)
				if got != tc.want {
					t.Errorf("putOutColumns failed. want %v, got %v", tc.want, got)
				}
			},
		)
	}
}

func TestPutOutColumns_AlwaysTrueWhenQueryHasColumns(t *testing.T) {
	query := &plugin.Query{
		Cmd:     metadata.CmdMany,
		Columns: []*plugin.Column{{}},
	}
	if putOutColumns(query) != true {
		t.Error("should be true when we have columns")
	}
}

func TestGetCursorPaginationSql(t *testing.T) {
	t.Run(
		"CursorPagination with one cursor field", func(t *testing.T) {
			query := Query{
				Cmd: metadata.CmdMany,
				Arg: QueryValue{
					Struct: &Struct{
						Fields: []Field{
							{
								Name:   "limit",
								DBName: "limit",
							},
							{
								Name:   "cursor",
								DBName: "id",
							},
						},
					},
				},
				SQL:              "SELECT id, name FROM table",
				Paginated:        true,
				CursorPagination: true,
			}

			sql := getCursorPaginationSql(
				query, []CursorField{
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "id",
							},
						},
					},
				},
			)

			expected := "SELECT cursor_pagination_source.* \nFROM (SELECT id, name FROM table) as cursor_pagination_source\nWHERE $0='' or  (id < $1)\nORDER BY id DESC\nLIMIT $1"
			if sql != expected {
				t.Errorf("getCursorPaginationSql failed. want %s, got %s", expected, sql)
			}
		},
	)

	t.Run(
		"CursorPagination with two cursor fields", func(t *testing.T) {
			query := Query{
				Cmd: metadata.CmdMany,
				Arg: QueryValue{
					Struct: &Struct{
						Fields: []Field{
							{
								Name:   "limit",
								DBName: "limit",
							},
							{
								Name:   "cursor",
								DBName: "id",
							},
						},
					},
				},
				SQL:              "SELECT id, name FROM table",
				Paginated:        true,
				CursorPagination: true,
			}

			sql := getCursorPaginationSql(
				query, []CursorField{
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "id",
							},
						},
					},
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "name",
							},
						},
					},
				},
			)

			expected := "SELECT cursor_pagination_source.* \nFROM (SELECT id, name FROM table) as cursor_pagination_source\nWHERE $0='' or  (id < $1 OR (id = $1 AND (name < $2)))\nORDER BY id DESC, name DESC\nLIMIT $1"
			if sql != expected {
				t.Errorf("getCursorPaginationSql failed. want %s, got %s", expected, sql)
			}
		},
	)

	t.Run(
		"CursorPagination with three cursor fields", func(t *testing.T) {
			query := Query{
				Cmd: metadata.CmdMany,
				Arg: QueryValue{
					Struct: &Struct{
						Fields: []Field{
							{
								Name:   "limit",
								DBName: "limit",
							},
							{
								Name:   "cursor",
								DBName: "id",
							},
						},
					},
				},
				SQL:              "SELECT id, name FROM table",
				Paginated:        true,
				CursorPagination: true,
			}

			sql := getCursorPaginationSql(
				query, []CursorField{
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "id",
							},
						},
					},
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "name",
							},
						},
					},
					{
						Field: Field{
							Column: &plugin.Column{
								Name: "created_at",
							},
						},
					},
				},
			)

			expected := "SELECT cursor_pagination_source.* \nFROM (SELECT id, name FROM table) as cursor_pagination_source\nWHERE $0='' or  (id < $1 OR (id = $1 AND (name < $2 OR (name = $2 AND (created_at < $3)))))\nORDER BY id DESC, name DESC, created_at DESC\nLIMIT $1"
			if sql != expected {
				t.Errorf("getCursorPaginationSql failed. want %s, got %s", expected, sql)
			}
		},
	)
}
