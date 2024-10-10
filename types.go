// types.go
package fsql

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Pagination struct {
	PageNo         int `json:"page_no"`
	ResultsPerPage int `json:"results_per_page"`
	PageMax        int `json:"page_max"`
	Count          int `json:"count"`
}

type CustomTime struct {
	Time  time.Time
	Valid bool
}

func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time, ct.Valid = time.Time{}, false
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		ct.Valid = true
	case string:
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return err
		}
		ct.Time = t
		ct.Valid = true
	default:
		return errors.New("invalid type for CustomTime")
	}
	return nil
}

func (ct CustomTime) Value() (driver.Value, error) {
	if !ct.Valid {
		return nil, nil
	}
	return ct.Time, nil
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if !ct.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(ct.Time.Format(time.RFC3339Nano))
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s != nil {
		t, err := time.Parse(time.RFC3339Nano, *s)
		if err != nil {
			return err
		}
		ct.Time = t
		ct.Valid = true
	} else {
		ct.Valid = false
	}
	return nil
}

type NullString struct {
	sql.NullString
}

func NewNullString(s string) NullString {
	if s == "" {
		return NullString{sql.NullString{String: "", Valid: false}}
	}
	return NullString{sql.NullString{String: s, Valid: true}}
}

type LocalizedText map[string]string

func (lt *LocalizedText) Scan(value interface{}) error {
	if value == nil {
		*lt = nil
		return nil
	}
	asBytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source was not []byte")
	}
	return json.Unmarshal(asBytes, lt)
}

func (lt LocalizedText) Value() (driver.Value, error) {
	if lt == nil {
		return nil, nil
	}
	return json.Marshal(lt)
}

type NullInt64 struct {
	sql.NullInt64
}

func NewNullInt64(s int64) NullInt64 {
	return NullInt64{sql.NullInt64{Int64: s, Valid: true}}
}

type NullBool struct {
	sql.NullBool
}

func NewNullBool(b bool) NullBool {
	return NullBool{sql.NullBool{Bool: b, Valid: true}}
}

type NullFloat64 struct {
	sql.NullFloat64
}

func NewNullFloat64(f float64) NullFloat64 {
	return NullFloat64{sql.NullFloat64{Float64: f, Valid: true}}
}
