package postgres

import (
	"fmt"
	"github.com/Kichiyaki/go-pg-logrus-query-logger/v10"
	"github.com/Kichiyaki/goutil/envutil"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/tw/twmodel"
)

var log = logrus.WithField("package", "pkg/postgres")

type Config struct {
	LogQueries bool
}

func Connect(cfg *Config) (*pg.DB, error) {
	db := pg.Connect(prepareOptions())

	if cfg != nil && cfg.LogQueries {
		db.AddQueryHook(querylogger.Logger{
			Log:            log,
			MaxQueryLength: 2000,
		})
	}

	if err := prepareDB(db); err != nil {
		return nil, err
	}

	return db, nil
}

func prepareOptions() *pg.Options {
	return &pg.Options{
		User:     envutil.GetenvString("DB_USER"),
		Password: envutil.GetenvString("DB_PASSWORD"),
		Database: envutil.GetenvString("DB_NAME"),
		Addr:     envutil.GetenvString("DB_HOST") + ":" + envutil.GetenvString("DB_PORT"),
		PoolSize: envutil.GetenvInt("DB_POOL_SIZE"),
	}
}

func prepareDB(db *pg.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "couldn't start a transaction")
	}
	defer func() {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrap(err, "prepareDB: couldn't rollback the transaction"))
		}
	}()

	dbModels := []interface{}{
		(*twmodel.SpecialServer)(nil),
		(*twmodel.Server)(nil),
		(*twmodel.Version)(nil),
		(*twmodel.PlayerToServer)(nil),
		(*twmodel.PlayerNameChange)(nil),
	}

	for _, model := range dbModels {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return errors.Wrap(err, "couldn't create the table")
		}
	}

	type statementWithParams struct {
		statement string
		params    []interface{}
	}

	for _, s := range []statementWithParams{
		{
			statement: pgDefaultValues,
		},
		{
			statement: allVersionsPGInsertStatements,
		},
		{
			statement: allSpecialServersPGInsertStatements,
		},
		{
			statement: pgDropSchemaFunctions,
			params:    []interface{}{pg.Safe("public")},
		},
		{
			statement: pgFunctions,
		},
	} {
		if _, err := tx.Exec(s.statement, s.params...); err != nil {
			return errors.Wrap(err, "couldn't prepare the db")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "couldn't commit changes")
	}

	var servers []*twmodel.Server
	if err := db.Model(&servers).Select(); err != nil {
		return errors.Wrap(err, "couldn't load servers")
	}

	for _, server := range servers {
		if err := createSchema(db, server, true); err != nil {
			return err
		}
	}

	return nil
}

func CreateSchema(db *pg.DB, server *twmodel.Server) error {
	return createSchema(db, server, false)
}

func SchemaExists(db *pg.DB, schemaName string) bool {
	exists, err := db.
		Model().
		Table("information_schema.schemata").
		Where("schema_name = ?", schemaName).
		Exists()
	if err != nil {
		return false
	}
	return exists
}

func createSchema(db *pg.DB, server *twmodel.Server, init bool) error {
	if !init && SchemaExists(db, server.Key) {
		return nil
	}

	tx, err := db.WithParam("SERVER", pg.Safe(server.Key)).Begin()
	if err != nil {
		return errors.Wrap(err, "couldn't start a transaction")
	}
	defer func() {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrap(err, "createSchema: Couldn't rollback the transaction"))
		}
	}()

	if _, err := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", server.Key)); err != nil {
		return errors.Wrap(err, "couldn't create for the server '"+server.Key+"'")
	}

	dbModels := []interface{}{
		(*twmodel.Tribe)(nil),
		(*twmodel.Player)(nil),
		(*twmodel.Village)(nil),
		(*twmodel.Ennoblement)(nil),
		(*twmodel.ServerStats)(nil),
		(*twmodel.TribeHistory)(nil),
		(*twmodel.PlayerHistory)(nil),
		(*twmodel.TribeChange)(nil),
		(*twmodel.DailyPlayerStats)(nil),
		(*twmodel.DailyTribeStats)(nil),
	}

	for _, model := range dbModels {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	statements := []string{
		serverPGFunctions,
		serverPGTriggers,
		serverPGDefaultValues,
	}
	if init {
		statements = append([]string{pgDropSchemaFunctions}, statements...)
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement, pg.Safe(server.Key), server.VersionCode); err != nil {
			return errors.Wrap(err, "couldn't initialize the schema")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "couldn't commit changes")
	}
	return nil
}
