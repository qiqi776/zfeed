package mysqltest

import (
	"errors"
	"testing"

	gomysql "github.com/go-sql-driver/mysql"
)

func TestIsDuplicateIndexErrorRecognizesMySQL1061(t *testing.T) {
	err := &gomysql.MySQLError{
		Number:  1061,
		Message: "Duplicate key name 'idx_content_root_status_list'",
	}

	if !isDuplicateIndexError(err) {
		t.Fatalf("isDuplicateIndexError(%v) = false, want true", err)
	}
}

func TestIsDuplicateIndexErrorRejectsOtherErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "nil",
			err:  nil,
		},
		{
			name: "generic error",
			err:  errors.New("boom"),
		},
		{
			name: "other mysql error",
			err: &gomysql.MySQLError{
				Number:  1062,
				Message: "Duplicate entry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isDuplicateIndexError(tt.err) {
				t.Fatalf("isDuplicateIndexError(%v) = true, want false", tt.err)
			}
		})
	}
}
