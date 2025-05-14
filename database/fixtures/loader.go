/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

// Package fixtures is based on https://github.com/go-testfixtures/testfixtures/tree/v2.6.0
package fixtures

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

type Context struct {
	db            *sql.DB
	fixturesFiles []*fixtureFile
}

type fixtureFile struct {
	path       string
	fileName   string
	content    []byte
	insertSQLs []insertSQL
}

type insertSQL struct {
	sql    string
	params []interface{}
}

func NewFolder(db *sql.DB, folderName string) (*Context, error) {
	fixtures, err := fixturesFromFolder(folderName)
	if err != nil {
		return nil, err
	}

	c, err := newContext(db, fixtures)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewFiles(db *sql.DB, fileNames ...string) (*Context, error) {
	fixtures, err := fixturesFromFiles(fileNames...)
	if err != nil {
		return nil, err
	}

	c, err := newContext(db, fixtures)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func newContext(db *sql.DB, fixtures []*fixtureFile) (*Context, error) {
	c := &Context{
		db:            db,
		fixturesFiles: fixtures,
	}

	if err := c.buildInsertSQLs(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Context) Load() error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, file := range c.fixturesFiles {
		if err = file.delete(tx); err != nil {
			return err
		}

		for _, i := range file.insertSQLs {
			if _, err = tx.Exec(i.sql, i.params...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (c *Context) buildInsertSQLs() error {
	for _, f := range c.fixturesFiles {
		var records []map[string]any
		if err := yaml.Unmarshal(f.content, &records); err != nil {
			return err
		}

		for _, record := range records {
			sqlText, values, err := f.buildInsertSQL(record)
			if err != nil {
				return err
			}

			f.insertSQLs = append(f.insertSQLs, insertSQL{sqlText, values})
		}
	}

	return nil
}

func (f *fixtureFile) fileNameWithoutExtension() string {
	return strings.Replace(f.fileName, filepath.Ext(f.fileName), "", 1)
}

func (f *fixtureFile) delete(tx *sql.Tx) error {
	_, err := tx.Exec(fmt.Sprintf("DELETE FROM `%s`", f.fileNameWithoutExtension()))
	return err
}

func (f *fixtureFile) buildInsertSQL(records map[string]any) (sqlStr string, params []interface{}, err error) {
	sqlColumns := make([]string, 0, len(records))

	for key, value := range records {
		sqlColumns = append(sqlColumns, fmt.Sprintf("`%s`", key))

		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "RAW=") {
				value, err = hex.DecodeString(strings.TrimPrefix(v, "RAW="))
				if err != nil {
					return "", nil, err
				}
			}
		case []any:
			b, err := json.Marshal(v)
			if err != nil {
				return "", nil, err
			}

			value = string(b)
		}

		params = append(params, value)
	}

	sqlStr = fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		f.fileNameWithoutExtension(),
		strings.Join(sqlColumns, ", "),
		strings.Join(slices.Repeat([]string{"?"}, len(sqlColumns)), ", "),
	)

	return
}

func fixturesFromFolder(folderName string) ([]*fixtureFile, error) {
	var files []*fixtureFile

	fileinfos, err := os.ReadDir(folderName)
	if err != nil {
		return nil, err
	}

	for _, fileinfo := range fileinfos {
		fileExt := filepath.Ext(fileinfo.Name())
		if !fileinfo.IsDir() && (fileExt == ".yml" || fileExt == ".yaml") {
			fixture := &fixtureFile{
				path:     path.Join(folderName, fileinfo.Name()),
				fileName: fileinfo.Name(),
			}

			fixture.content, err = os.ReadFile(fixture.path)
			if err != nil {
				return nil, err
			}

			files = append(files, fixture)
		}
	}

	return files, nil
}

func fixturesFromFiles(fileNames ...string) ([]*fixtureFile, error) {
	var err error

	fixtureFiles := make([]*fixtureFile, 0, len(fileNames))

	for _, f := range fileNames {
		fixture := &fixtureFile{
			path:     f,
			fileName: filepath.Base(f),
		}

		fixture.content, err = os.ReadFile(fixture.path)
		if err != nil {
			return nil, err
		}

		fixtureFiles = append(fixtureFiles, fixture)
	}

	return fixtureFiles, nil
}
