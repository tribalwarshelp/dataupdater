package db

import (
	"fmt"
	gopglogrusquerylogger "github.com/Kichiyaki/go-pg-logrus-query-logger/v10"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/models"

	envutils "github.com/tribalwarshelp/cron/utils/env"
)

var log = logrus.WithField("package", "db")

type Config struct {
	LogQueries bool
}

func New(cfg *Config) (*pg.DB, error) {
	db := pg.Connect(prepareOptions())

	if cfg != nil && cfg.LogQueries {
		db.AddQueryHook(gopglogrusquerylogger.QueryLogger{
			Entry:          log,
			MaxQueryLength: 5000,
		})
	}

	if err := prepareDB(db); err != nil {
		return nil, errors.Wrap(err, "New")
	}

	return db, nil
}

func prepareOptions() *pg.Options {
	return &pg.Options{
		User:     envutils.GetenvString("DB_USER"),
		Password: envutils.GetenvString("DB_PASSWORD"),
		Database: envutils.GetenvString("DB_NAME"),
		Addr:     envutils.GetenvString("DB_HOST") + ":" + envutils.GetenvString("DB_PORT"),
		PoolSize: envutils.GetenvInt("DB_POOL_SIZE"),
	}
}

func prepareDB(db *pg.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "Couldn't start a transaction")
	}
	defer func() {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrap(err, "prepareDB: Couldn't rollback the transaction"))
		}
	}()

	dbModels := []interface{}{
		(*models.SpecialServer)(nil),
		(*models.Server)(nil),
		(*models.Version)(nil),
		(*models.PlayerToServer)(nil),
		(*models.PlayerNameChange)(nil),
	}

	for _, model := range dbModels {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return errors.Wrap(err, "Couldn't create the table")
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
			return errors.Wrap(err, "Couldn't initialize the db")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Couldn't commit changes")
	}

	var servers []*models.Server
	if err := db.Model(&servers).Select(); err != nil {
		return errors.Wrap(err, "Couldn't load servers")
	}

	for _, server := range servers {
		if err := createSchema(db, server, true); err != nil {
			return err
		}
	}

	return nil
}

func CreateSchema(db *pg.DB, server *models.Server) error {
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

func createSchema(db *pg.DB, server *models.Server, init bool) error {
	if !init && SchemaExists(db, server.Key) {
		return nil
	}

	tx, err := db.WithParam("SERVER", pg.Safe(server.Key)).Begin()
	if err != nil {
		return errors.Wrap(err, "CreateSchema: couldn't start a transaction")
	}
	defer func() {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrap(err, "createSchema: Couldn't rollback the transaction"))
		}
	}()

	if _, err := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", server.Key)); err != nil {
		return errors.Wrap(err, "CreateSchema: couldn't create the schema")
	}

	dbModels := []interface{}{
		(*models.Tribe)(nil),
		(*models.Player)(nil),
		(*models.Village)(nil),
		(*models.Ennoblement)(nil),
		(*models.ServerStats)(nil),
		(*models.TribeHistory)(nil),
		(*models.PlayerHistory)(nil),
		(*models.TribeChange)(nil),
		(*models.DailyPlayerStats)(nil),
		(*models.DailyTribeStats)(nil),
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
			return errors.Wrap(err, "CreateSchema: couldn't initialize the schema")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "CreateSchema: couldn't commit changes")
	}
	return nil
}
