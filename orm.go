// orm.go
package fsql

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Join struct {
	Table       string
	TableAlias  string
	JoinType    string
	OnCondition string
}

type QueryBuilder struct {
	Table string
	Joins []Join
}

func GetInsertQuery(tableName string, valuesMap map[string]interface{}, returning string) (string, []interface{}) {
	_, fields := GetInsertFields(tableName)
	defaultValues := GetInsertValues(tableName)

	var columns []string
	var placeholders []string
	var queryValues []interface{}
	counter := 1

	for _, field := range fields {
		columns = append(columns, field)
		if val, ok := valuesMap[field]; ok {
			placeholders = append(placeholders, fmt.Sprintf("$%d", counter))
			queryValues = append(queryValues, val)
			counter++
		} else if defVal, ok := defaultValues[field]; ok {
			if isSQLFunction(defVal) {
				placeholders = append(placeholders, defVal)
			} else {
				placeholders = append(placeholders, fmt.Sprintf("$%d", counter))
				queryValues = append(queryValues, defVal)
				counter++
			}
		} else {
			placeholders = append(placeholders, "DEFAULT")
		}
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`, tableName, strings.Join(columns, ","), strings.Join(placeholders, ","))
	if returning != "" {
		query += fmt.Sprintf(` RETURNING "%s"`, returning)
	}
	return query, queryValues
}

func isSQLFunction(value string) bool {
	functions := map[string]struct{}{
		"NOW()":   {},
		"NULL":    {},
		"true":    {},
		"false":   {},
		"DEFAULT": {},
	}
	_, exists := functions[value]
	return exists
}

func GetUpdateQuery(tableName string, valuesMap map[string]interface{}, returning string) (string, []interface{}) {
	_, fields := GetUpdateFields(tableName)

	var setClauses []string
	var queryValues []interface{}
	counter := 1

	for _, field := range fields {
		if value, exists := valuesMap[field]; exists {
			setClauses = append(setClauses, fmt.Sprintf(`%s = $%d`, field, counter))
			queryValues = append(queryValues, value)
			counter++
		}
	}

	if len(setClauses) == 0 {
		panic("No fields to update")
	}

	primaryKey, pkExists := valuesMap[returning]
	if !pkExists {
		panic("Primary key not found in valuesMap")
	}

	query := fmt.Sprintf(`UPDATE "%s" SET %s WHERE "%s" = $%d RETURNING "%s"`, tableName, strings.Join(setClauses, ", "), returning, counter, returning)
	queryValues = append(queryValues, primaryKey)

	return query, queryValues
}

func SelectBase(table string, alias string) *QueryBuilder {
	return &QueryBuilder{
		Table: table,
		Joins: []Join{},
	}
}

func (qb *QueryBuilder) Left(table string, alias string, on string) *QueryBuilder {
	qb.Joins = append(qb.Joins, Join{
		Table:       table,
		TableAlias:  alias,
		JoinType:    "LEFT JOIN",
		OnCondition: on,
	})
	return qb
}

func (qb *QueryBuilder) Build() string {
	fieldsArray, _ := GetSelectFields(qb.Table, "")
	fields := strings.Join(fieldsArray, ",")

	for _, join := range qb.Joins {
		joinFields, _ := GetSelectFields(join.Table, join.TableAlias)
		fields += ", " + strings.Join(joinFields, ",")
	}

	var joins []string
	for _, join := range qb.Joins {
		table := fmt.Sprintf(`"%s"`, join.Table)
		if join.TableAlias != "" {
			table = fmt.Sprintf(`"%s" AS "%s"`, join.Table, join.TableAlias)
		}
		joins = append(joins, fmt.Sprintf(`%s %s ON %s`, join.JoinType, table, join.OnCondition))
	}

	return fmt.Sprintf(`SELECT %s FROM "%s" %s`, fields, qb.Table, strings.Join(joins, " "))
}

func GenNewUUID() string {
	return uuid.New().String()
}
