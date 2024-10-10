// types.go
package fsql

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

type Pagination struct {
	PageNo         int `json:"page_no"`
	ResultsPerPage int `json:"results_per_page"`
	PageMax        int `json:"page_max"`
	Count          int `json:"count"`
}

type CustomTime struct {
	sql.NullTime
}

type TimeResponse struct {
	ISO    string `json:"iso"`
	TZ     string `json:"tz"`
	Unix   int64  `json:"unix"`
	UnixMS int64  `json:"unixms"`
	US     int64  `json:"us"`
	Full   int64  `json:"full,omitempty,string"`
}

func NewCustomTimeNull() *CustomTime {
	return &CustomTime{
		NullTime: sql.NullTime{
			Time:  time.Time{},
			Valid: false,
		},
	}
}

func NewCustomTime(t time.Time) *CustomTime {
	return &CustomTime{
		NullTime: sql.NullTime{
			Time:  t,
			Valid: true,
		},
	}
}

func NewCustomTimeInt64(int64Time int64) *CustomTime {
	return &CustomTime{
		NullTime: sql.NullTime{
			Time:  time.Unix(0, int64Time*int64(time.Millisecond)),
			Valid: true,
		},
	}
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
		return nil
	case TimeResponse:
		// Parsing time from TimeResponse
		t, err := time.Parse(time.RFC3339Nano, v.ISO)
		if err != nil {
			return err
		}
		ct.Time = t
		ct.Valid = true
		return nil
	case *TimeResponse:
		// Parsing time from TimeResponse
		t, err := time.Parse(time.RFC3339Nano, v.ISO)
		if err != nil {
			return err
		}
		ct.Time = t
		ct.Valid = true
		return nil
	case int64:
		ct.Time = time.Unix(0, v*int64(time.Millisecond))
		ct.Valid = true
		return nil
	case string:
		// Support for the format: "1661-06-21"
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
		ct.Time = t
		ct.Valid = true
		return nil
	default:
		return ct.NullTime.Scan(value)
	}
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
	timezone := ct.Time.Location().String()
	if timezone == "" {
		timezone = "UTC"
	}

	tr := TimeResponse{
		ISO:    ct.Time.Format(time.RFC3339Nano),
		TZ:     timezone,
		Unix:   ct.Time.Unix(),
		UnixMS: ct.Time.UnixMilli(),
		US:     int64(ct.Time.Nanosecond()),
		Full:   ct.Time.UnixMicro(),
	}

	return json.Marshal(tr)
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	var t TimeResponse
	if err := json.Unmarshal(b, &t); err != nil {
		var ts string
		if err := json.Unmarshal(b, &ts); err != nil {
			return err
		}
		parsedTime, err := time.Parse("2006-01-02", ts)
		if err != nil {
			return err
		}
		ct.Time = parsedTime
		ct.Valid = true
		return nil
	}

	if t.ISO == "" {
		ct.Valid = false
		ct.Time = time.Time{}
	} else {
		parsedTime, err := time.Parse(time.RFC3339Nano, t.ISO)
		if err != nil {
			return err
		}
		ct.Time = parsedTime
		ct.Valid = true
	}

	return nil
}

type NullString struct {
	sql.NullString
}

func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	return ns.NullString.Scan(value)
}

func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.String)
	}
	return json.Marshal(nil)
}

func (ns *NullString) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s != nil {
		ns.Valid = true
		ns.String = *s
	} else {
		ns.Valid = false
	}

	return nil
}

func NewNullString(s string) *NullString {
	valid := false
	if s != "" {
		valid = true
	}
	return &NullString{sql.NullString{String: s, Valid: valid}}
}

type LocalizedText map[string]string

// Scan implements the sql.Scanner interface.
func (lt *LocalizedText) Scan(value interface{}) error {
	if value == nil {
		*lt = nil
		return nil
	}
	asBytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source was not []byte")
	}
	err := json.Unmarshal(asBytes, lt)
	if err != nil {
		return err
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (lt LocalizedText) Value() (driver.Value, error) {
	if lt == nil {
		return nil, nil
	}
	return json.Marshal(lt)
}

func (lt *LocalizedText) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*lt = m
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (lt LocalizedText) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(lt))
}

type NullInt64 struct {
	sql.NullInt64
}

func (ns *NullInt64) Scan(value interface{}) error {
	if value == nil {
		ns.Int64, ns.Valid = 0, false
		return nil
	}
	switch v := value.(type) {
	case int:
		ns.Int64 = int64(v)
		ns.Valid = true
		return nil
	case int64:
		ns.Int64 = v
		ns.Valid = true
		return nil
	case float64:
		ns.Int64 = int64(v)
		ns.Valid = true
		return nil
	case string:
		intValue, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			ns.Valid = false
			ns.Int64 = 0
			return nil
		}
		ns.Int64 = intValue
		ns.Valid = true
		return nil
	}
	return ns.NullInt64.Scan(value)
}

func (ns NullInt64) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.Int64, nil
}

func (ns NullInt64) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Int64)
	}
	return json.Marshal(nil)
}

func (ns *NullInt64) UnmarshalJSON(b []byte) error {
	var s *int64
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s != nil {
		ns.Valid = true
		ns.Int64 = *s
	} else {
		ns.Valid = false
	}

	return nil
}

func NewNullInt64FromString(s string) *NullInt64 {
	if s == "" {
		return &NullInt64{sql.NullInt64{Int64: 0, Valid: false}}
	}
	intValue, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &NullInt64{sql.NullInt64{Int64: 0, Valid: false}}
	}
	return &NullInt64{sql.NullInt64{Int64: intValue, Valid: true}}
}

func NewNullInt64(s int64) *NullInt64 {
	return &NullInt64{sql.NullInt64{Int64: s, Valid: true}}
}

type NullBool struct {
	sql.NullBool
}

func (ns *NullBool) Scan(value interface{}) error {
	if value == nil {
		ns.Bool, ns.Valid = false, false
		return nil
	}
	return ns.NullBool.Scan(value)
}

func (ns NullBool) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.Bool, nil
}

func (ns NullBool) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Bool)
	}
	return json.Marshal(nil)
}

func (ns *NullBool) UnmarshalJSON(b []byte) error {
	var s *bool
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s != nil {
		ns.Valid = true
		ns.Bool = *s
	} else {
		ns.Valid = false
	}

	return nil
}

func NewNullBoolFromString(s string) *NullBool {
	valid := false
	if s != "" {
		valid = true
	}
	boolValue, err := strconv.ParseBool(s)
	if err != nil {
		return &NullBool{sql.NullBool{Bool: false, Valid: false}}
	}
	return &NullBool{sql.NullBool{Bool: boolValue, Valid: valid}}
}

func NewNullBool(s bool) *NullBool {
	return &NullBool{sql.NullBool{Bool: s, Valid: true}}
}

type NullFloat64 struct {
	sql.NullFloat64
}

func (ns *NullFloat64) Scan(value interface{}) error {
	if value == nil {
		ns.Float64, ns.Valid = 0, false
		return nil
	}
	return ns.NullFloat64.Scan(value)
}

func (ns NullFloat64) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.Float64, nil
}

func (ns NullFloat64) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Float64)
	}
	return json.Marshal(nil)
}

func (ns *NullFloat64) UnmarshalJSON(b []byte) error {
	var s *float64
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s != nil {
		ns.Valid = true
		ns.Float64 = *s
	} else {
		ns.Valid = false
	}

	return nil
}

func NewNullFloat64FromString(s string) *NullFloat64 {
	if s == "" {
		return &NullFloat64{sql.NullFloat64{Float64: 0, Valid: false}}
	}
	floatValue, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return &NullFloat64{sql.NullFloat64{Float64: 0, Valid: false}}
	}
	return &NullFloat64{sql.NullFloat64{Float64: floatValue, Valid: true}}
}

func NewNullFloat64(s float64) *NullFloat64 {
	return &NullFloat64{sql.NullFloat64{Float64: s, Valid: true}}
}

type IntDictionary map[string]int

// Scan implements the sql.Scanner interface.
func (id *IntDictionary) Scan(value interface{}) error {
	asBytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source was not []byte")
	}
	err := json.Unmarshal(asBytes, id)
	if err != nil {
		return err
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (id IntDictionary) Value() (driver.Value, error) {
	return json.Marshal(id)
}

func (id *IntDictionary) UnmarshalJSON(data []byte) error {
	var m map[string]int
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*id = m
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (id IntDictionary) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]int(id))
}
