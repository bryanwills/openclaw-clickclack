package postgres

import "testing"

func TestShouldRedactExportColumn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		table  string
		column string
		want   bool
	}{
		{table: "auth_magic_links", column: "token", want: true},
		{table: "auth_magic_links", column: "token_hash", want: true},
		{table: "sessions", column: "token", want: true},
		{table: "sessions", column: "token_hash", want: true},
		{table: "bot_tokens", column: "token_hash", want: true},
		{table: "uploads", column: "storage_path", want: true},
		{table: "uploads", column: "filename", want: false},
		{table: "users", column: "email", want: false},
	}
	for _, test := range tests {
		if got := shouldRedactExportColumn(test.table, test.column); got != test.want {
			t.Fatalf("shouldRedactExportColumn(%q, %q) = %v, want %v", test.table, test.column, got, test.want)
		}
	}
}
