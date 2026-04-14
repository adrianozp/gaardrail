package sql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/adrianozp/gaardrail/app/entities"
	sqlrepo "github.com/adrianozp/gaardrail/app/repositories/targets/sql"
	sqlclient "github.com/adrianozp/gaardrail/internal/sqlclient"
	"github.com/stretchr/testify/require"
)

func TestSQLRepository_Push_ExecutesQueryFromBody(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	query := "INSERT INTO events (id) VALUES ('abc')"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))

	client := sqlclient.NewFromDB(db)
	repo := sqlrepo.NewSQLRepository(client)

	err = repo.Push(context.Background(), entities.Message{ID: "abc", Body: []byte(query)})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLRepository_Push_ReturnsErrorOnQueryFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(".*").WillReturnError(fmt.Errorf("db error"))

	client := sqlclient.NewFromDB(db)
	repo := sqlrepo.NewSQLRepository(client)

	err = repo.Push(context.Background(), entities.Message{ID: "1", Body: []byte("SELECT 1")})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
