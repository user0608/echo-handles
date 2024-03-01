package echohandles

import (
	"database/sql"
	"database/sql/driver"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/user0608/goones/answer"
	"github.com/user0608/goones/errs"
	"gorm.io/gorm"
)

// page es el numero de pagina y perPage es el numero de registros por pagina
// aparter de esa informacion se calcula el offset y limit
// el maximo de registros por pagina es 1000
func applyLimitOffset(tx *gorm.DB, page int64, perPage int64) *gorm.DB {
	offset := (page - 1) * perPage
	return tx.Offset(int(offset)).Limit(int(perPage))
}

func getPagination(c echo.Context) (int64, int64) {
	page := int64(1)
	perPage := int64(10)
	var err error
	if p := c.QueryParam("page"); p != "" {
		page, err = strconv.ParseInt(p, 10, 64)
		if err != nil {
			page = 1
			slog.Warn("getPagination", "error", err)
		}
	}
	if pp := c.QueryParam("perPage"); pp != "" {
		perPage, err = strconv.ParseInt(pp, 10, 64)
		if err != nil {
			perPage = 10
			slog.Warn("getPagination", "error", err)
		}
	}
	return page, perPage
}
func getLimitOffset(c echo.Context) (int64, int64) {
	var limit = int64(10)
	var offset = int64(0)
	var err error
	if l := c.QueryParam("limit"); l != "" {
		limit, err = strconv.ParseInt(l, 10, 64)
		if err != nil {
			limit = 1
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		offset, err = strconv.ParseInt(o, 10, 64)
		if err != nil {
			limit = 1
		}
	}
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func TableQueryHandle(tables []string, tx *gorm.DB) echo.HandlerFunc {
	var tableMap = make(map[string]bool)
	for _, table := range tables {
		tableMap[table] = true
	}
	return func(c echo.Context) error {
		var table = c.Param("table")
		var count int64
		if err := tx.Table(table).Count(&count).Error; err != nil {
			slog.Error("GetTableHandle", "error", err)
			return answer.Err(c, errs.Internal(errs.ErrInternal))
		}
		if _, ok := tableMap[table]; !ok {
			return answer.Err(c, errs.Bad("table not found"))
		}
		page, perPage := getPagination(c)
		if page < 1 {
			page = 1
		}
		if perPage < 1 {
			perPage = 10
		}
		rows, err := applyLimitOffset(tx.Table(table), page, perPage).Rows()
		if err != nil {
			slog.Error("GetTableHandle", "error", err)
			return answer.Err(c, errs.Internal(errs.ErrInternal))
		}

		result, err := prepareRecords(rows)
		if err != nil {
			return answer.Err(c, err)
		}
		return answer.OKPage(c, page, perPage, count, result)
	}
}

func TableQueryWithLimitOffsetHandle(tables []string, tx *gorm.DB) echo.HandlerFunc {
	var tableMap = make(map[string]bool)
	for _, table := range tables {
		tableMap[table] = true
	}
	return func(c echo.Context) error {
		var table = c.Param("table")
		var count int64
		if err := tx.Table(table).Count(&count).Error; err != nil {
			slog.Error("GetTableHandle", "error", err)
			return answer.Err(c, errs.Internal(errs.ErrInternal))
		}
		limit, offset := getLimitOffset(c)
		rows, err := tx.Offset(int(offset)).Limit(int(limit)).Table(table).Rows()
		if err != nil {
			slog.Error("GetTableHandle", "error", err)
			return answer.Err(c, errs.Internal(errs.ErrInternal))
		}
		result, err := prepareRecords(rows)
		if err != nil {
			return answer.Err(c, err)
		}
		return answer.OKLimitOffset(c, limit, offset, count, result)
	}
}

func TableQueryWithoutPaginationHandle(tables []string, tx *gorm.DB) echo.HandlerFunc {
	var tableMap = make(map[string]bool)
	for _, table := range tables {
		tableMap[table] = true
	}
	return func(c echo.Context) error {
		var table = c.Param("table")
		if _, ok := tableMap[table]; !ok {
			return answer.Err(c, errs.Bad("table not found"))
		}
		rows, err := tx.Table(table).Rows()
		if err != nil {
			slog.Error("GetTableHandle", "error", err)
			return answer.Err(c, errs.Internal(errs.ErrInternal))
		}
		result, err := prepareRecords(rows)
		if err != nil {
			return answer.Err(c, err)
		}
		return answer.Ok(c, result)
	}
}

func prepareRecords(rows *sql.Rows) ([]JsonObject, error) {
	columns, err := rows.Columns()
	if err != nil {
		slog.Error("GetTableHandle", "error", err)
		return nil, errs.Internal(errs.ErrInternal)
	}
	for i, column := range columns {
		columns[i] = columnName(column)
	}
	var result = make([]JsonObject, 0)
	for rows.Next() {
		var values = make([]interface{}, len(columns))
		var pointers = make([]interface{}, len(columns))
		for i := range pointers {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			slog.Error("GetTableHandle", "error", err)
			return nil, errs.Internal(errs.ErrInternal)
		}
		var row = scanJsonField(values, columns)
		result = append(result, row)

	}
	return result, nil
}

func scanJsonField(values []interface{}, columns []string) JsonObject {
	var jsonObject JsonObject
	for idx, column := range columns {
		var item = JsonField{Field: column}
		if reflectValue := reflect.Indirect(reflect.Indirect(reflect.ValueOf(values[idx]))); reflectValue.IsValid() {
			item.Value = reflectValue.Interface()
			if valuer, ok := item.Value.(driver.Valuer); ok {
				item.Value, _ = valuer.Value()
			} else if b, ok := item.Value.(sql.RawBytes); ok {
				item.Value = string(b)
			}
		} else {
			item.Value = nil
		}
		item.Value = fixUint8AFloat64(item.Value)
		jsonObject = append(jsonObject, item)
	}
	return jsonObject
}

func fixUint8AFloat64(v any) any {
	switch v := v.(type) {
	case []byte:
		return string(v)
	}
	return v
}
