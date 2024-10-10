// fsql.go
package fsql

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

var Db *sqlx.DB

func InitDB(database string) {
	var err error
	Db, err = sqlx.Connect("postgres", database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Set reasonable limits
	Db.SetMaxOpenConns(25)
	Db.SetMaxIdleConns(25)
	Db.SetConnMaxLifetime(5 * time.Minute)
}

// CloseDB closes the database connection
func CloseDB() {
	if Db != nil {
		if err := Db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
