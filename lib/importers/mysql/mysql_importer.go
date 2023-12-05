package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/common"
	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type Importer struct {
	console consoles.Console
	storage storages.Storage
}

func NewImporter(console consoles.Console, storage storages.Storage) *Importer {
	return &Importer{
		console: console,
		storage: storage,
	}
}

func (i *Importer) Import(connectionString string) error {
	projects, err := i.storage.LoadProjects()
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return errors.Wrapf(err, "error connecting to MySQL using %v", connectionString)
	}

	defer db.Close()

	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	err = i.importTables(db, projects)
	if err != nil {
		return err
	}

	err = i.importFKs(db, projects)
	if err != nil {
		return err
	}

	fmt.Printf("Writing results...\n")

	return i.storage.WriteProjects()
}

func (i *Importer) importTables(db *sql.DB, projs *model.Projects) error {
	results, err := db.Query(`
		select TABLE_SCHEMA schema_name,
			   TABLE_NAME   table_name,
			   TABLE_ROWS   rows,
			   DATA_LENGTH  data_size,
			   INDEX_LENGTH index_size
		from information_schema.TABLES
		where TABLE_TYPE = 'BASE TABLE'
		  and TABLE_SCHEMA <> 'information_schema'
		`)
	if err != nil {
		return errors.Wrap(err, "error querying database tables")
	}

	type tableInfo struct {
		schemaName string
		tableName  string
		rows       int
		dataSize   int
		indexSize  int
	}

	var changedProjs []*model.Project

	for results.Next() {
		var table tableInfo

		err = results.Scan(&table.schemaName, &table.tableName, &table.rows, &table.dataSize, &table.indexSize)
		if err != nil {
			return errors.Wrap(err, "error querying database tables")
		}

		fmt.Printf("Importing table %v.%v (%v data, %v indexes)\n", table.schemaName, table.tableName,
			humanize.Bytes(uint64(table.dataSize)), humanize.Bytes(uint64(table.indexSize)))

		proj := projs.GetOrCreate(table.tableName)
		proj.Groups = []string{table.schemaName}
		proj.Type = model.DatabaseType

		proj.AddSize("table", &model.Size{
			Lines: table.rows,
			Bytes: table.dataSize + table.indexSize,
			Other: map[string]int{
				"data":    table.dataSize,
				"indexes": table.indexSize,
			},
		})

		changedProjs = append(changedProjs, proj)
	}

	common.CreateTableNameParts(changedProjs)

	return nil
}

func (i *Importer) importFKs(db *sql.DB, projs *model.Projects) error {
	results, err := db.Query(`
		select CONSTRAINT_SCHEMA schema_name,
			   TABLE_NAME,
			   REFERENCED_TABLE_NAME
		from information_schema.REFERENTIAL_CONSTRAINTS
		`)
	if err != nil {
		return errors.Wrap(err, "error querying database FKs")
	}

	type fkInfo struct {
		schemaName          string
		tableName           string
		referencedTableName string
	}

	type rootAndName struct {
		root string
		name string
	}
	toSave := map[rootAndName]bool{}

	for results.Next() {
		var fk fkInfo

		err = results.Scan(&fk.schemaName, &fk.tableName, &fk.referencedTableName)
		if err != nil {
			return errors.Wrap(err, "error querying database FKs")
		}

		fmt.Printf("Importing dependency %v.%v => %v.%v\n",
			fk.schemaName, fk.tableName, fk.schemaName, fk.referencedTableName)

		proj := projs.GetOrCreate(fk.tableName)

		dep := projs.GetOrCreate(fk.referencedTableName)
		proj.GetOrCreateDependency(dep)

		toSave[rootAndName{fk.schemaName, fk.tableName}] = true
	}

	return nil
}
