// cache.go
package fsql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/soulkyn-ai/nyxutils"
)

var modelFieldsCache = nyxutils.NewSafeMap[*modelInfo]()

type modelInfo struct {
	dbTagMap          map[string]string
	dbInsertValueMap  map[string]string
	dbFieldsSelect    []string
	dbFieldsInsert    []string
	dbFieldsUpdate    []string
	dbFieldsSelectMap map[string]struct{}
	dbFieldsInsertMap map[string]struct{}
	dbFieldsUpdateMap map[string]struct{}
}

// InitModelTagCache initializes the model metadata cache
func InitModelTagCache(model interface{}, tableName string) {
	if _, exists := getModelInfo(tableName); exists {
		return // Already initialized
	}

	modelType := getModelType(model)

	dbTagMap := make(map[string]string)
	dbInsertValueMap := make(map[string]string)
	var dbFieldsSelect, dbFieldsInsert, dbFieldsUpdate []string
	dbFieldsSelectMap := make(map[string]struct{})
	dbFieldsInsertMap := make(map[string]struct{})
	dbFieldsUpdateMap := make(map[string]struct{})

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		dbTagValue := field.Tag.Get("db")
		if dbTagValue == "" || dbTagValue == "-" {
			continue
		}

		dbMode := field.Tag.Get("dbMode")
		dbInsertValue := field.Tag.Get("dbInsertValue")
		modes := strings.Split(dbMode, ",")

		modeFlags := make(map[string]bool)
		for _, mode := range modes {
			modeFlags[mode] = true
		}

		if modeFlags["s"] || modeFlags["link"] {
			continue
		}

		dbTagMap[field.Name] = dbTagValue

		if modeFlags["i"] {
			dbFieldsInsert = append(dbFieldsInsert, dbTagValue)
			dbFieldsInsertMap[dbTagValue] = struct{}{}
			if dbInsertValue != "" {
				dbInsertValueMap[dbTagValue] = dbInsertValue
			}
		}
		if modeFlags["u"] {
			dbFieldsUpdate = append(dbFieldsUpdate, dbTagValue)
			dbFieldsUpdateMap[dbTagValue] = struct{}{}
		}
		if !modeFlags["link"] && !modeFlags["s"] {
			dbFieldsSelect = append(dbFieldsSelect, dbTagValue)
			dbFieldsSelectMap[dbTagValue] = struct{}{}
		}
	}

	modelInfo := &modelInfo{
		dbTagMap:          dbTagMap,
		dbInsertValueMap:  dbInsertValueMap,
		dbFieldsSelect:    dbFieldsSelect,
		dbFieldsInsert:    dbFieldsInsert,
		dbFieldsUpdate:    dbFieldsUpdate,
		dbFieldsSelectMap: dbFieldsSelectMap,
		dbFieldsInsertMap: dbFieldsInsertMap,
		dbFieldsUpdateMap: dbFieldsUpdateMap,
	}

	modelFieldsCache.Set(tableName, modelInfo)
}

func getModelInfo(tableName string) (*modelInfo, bool) {
	if modelInfo, ok := modelFieldsCache.Get(tableName); ok {
		return modelInfo, true
	}
	return nil, false
}

func getModelType(model interface{}) reflect.Type {
	modelType := reflect.TypeOf(model)
	for modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected a struct, got %s", modelType.Kind()))
	}

	return modelType
}

func getFieldsByMode(tableName, mode, aliasTableName string) ([]string, []string) {
	modelInfo, ok := getModelInfo(tableName)
	if !ok {
		panic("table name not initialized: " + tableName)
	}

	var fields []string
	var fieldNames []string
	var dbFields []string

	switch mode {
	case "select":
		dbFields = modelInfo.dbFieldsSelect
	case "insert":
		dbFields = modelInfo.dbFieldsInsert
	case "update":
		dbFields = modelInfo.dbFieldsUpdate
	default:
		panic("invalid mode")
	}

	for _, fieldName := range dbFields {
		if aliasTableName != "" {
			alias := strings.ReplaceAll(aliasTableName, `"`, "")
			fields = append(fields, fmt.Sprintf(`"%s".%s AS "%s.%s"`, alias, fieldName, alias, fieldName))
		} else {
			fields = append(fields, fmt.Sprintf(`"%s".%s`, tableName, fieldName))
		}
		fieldNames = append(fieldNames, fieldName)
	}

	return fields, fieldNames
}

// Public API functions
func GetSelectFields(tableName, aliasTableName string) ([]string, []string) {
	return getFieldsByMode(tableName, "select", aliasTableName)
}

func GetInsertFields(tableName string) ([]string, []string) {
	return getFieldsByMode(tableName, "insert", "")
}

func GetUpdateFields(tableName string) ([]string, []string) {
	return getFieldsByMode(tableName, "update", "")
}

func GetInsertValues(tableName string) map[string]string {
	modelInfo, ok := getModelInfo(tableName)
	if !ok {
		panic("table name not initialized: " + tableName)
	}
	return modelInfo.dbInsertValueMap
}
