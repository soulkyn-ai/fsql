// fsql_test.go
package fsql

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type AIModelTest struct {
	UUID                  NullString `json:"UUID" db:"uuid" dbMode:"i"`
	Key                   NullString `json:"Key" db:"key" dbMode:"i,u"`
	Name                  NullString `json:"Name" db:"name" dbMode:"i,u" dbInsertValue:"NULL"`
	Description           NullString `json:"Description" db:"description" dbMode:"i,u" dbInsertValue:"NULL"`
	Type                  NullString `json:"Type" db:"type" dbMode:"i,u"`
	Provider              NullString `json:"Provider" db:"provider" dbMode:"i,u"`
	Settings              NullString `json:"Settings" db:"settings" dbMode:"i,u" dbInsertValue:"NULL"`
	DefaultNegativePrompt NullString `json:"DefaultNegativePrompt" db:"default_negative_prompt" dbMode:"i,u" dbInsertValue:"NULL"`
}
type RealmTest struct {
	UUID      string      `json:"UUID" db:"uuid" dbMode:"i"`
	CreatedAt *CustomTime `json:"CreatedAt" db:"created_at" dbMode:"i" dbInsertValue:"NOW()"`
	UpdatedAt *CustomTime `json:"UpdatedAt" db:"updated_at" dbMode:"i,u" dbInsertValue:"NOW()"`
	Name      string      `json:"Name" db:"name" dbMode:"i,u"`
}

type WebsiteTest struct {
	UUID      string     `json:"UUID" db:"uuid" dbMode:"i"`
	CreatedAt CustomTime `json:"CreatedAt" db:"created_at" dbMode:"i" dbInsertValue:"NOW()"`
	UpdatedAt CustomTime `json:"UpdatedAt" db:"updated_at" dbMode:"i" dbInsertValue:"NOW()"`
	Domain    string     `json:"Domain" db:"domain" dbMode:"i,u"`
	Realm     *RealmTest `json:"Realm,omitempty" db:"r" dbMode:"l"`
	RealmUUID string     `json:"RealmUUID" db:"realm_uuid" dbMode:"i"`
}

var (
	aiModelBaseQuery       string
	realmQuerySelectBase   string
	websiteQuerySelectBase string
)

func TestMain(m *testing.M) {
	// Define command-line flags
	dbUser := flag.String("dbuser", "test_user", "Database user")
	dbPassword := flag.String("dbpass", "test_password", "Database password")
	dbName := flag.String("dbname", "test_db", "Database name")
	dbHost := flag.String("dbhost", "localhost", "Database host")
	dbPort := flag.String("dbport", "5432", "Database port")

	// Parse flags
	flag.Parse()

	// Initialize model cache
	InitAIModel()
	initRealmModel()
	initWebsiteModel()

	// Initialize the database connection
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)
	InitDB(connStr)

	// Run tests
	code := m.Run()

	// Clean up after tests
	if err := cleanDatabase(); err != nil {
		log.Fatalf("Failed to clean database: %v", err)
	}

	Db.Close()
	os.Exit(code)
}

func InitAIModel() {
	InitModelTagCache(AIModelTest{}, "ai_model")
	aiModelBaseQuery = SelectBase("ai_model", "").Build()
}

func initRealmModel() {
	InitModelTagCache(RealmTest{}, "realm")
	realmQuerySelectBase = SelectBase("realm", "realm").Build()
}

func initWebsiteModel() {
	InitModelTagCache(WebsiteTest{}, "website")
	websiteQuerySelectBase = SelectBase("website", "website").Left("realm", "r", "website.realm_uuid = r.uuid").Build()
}

func cleanDatabase() error {
	_, err := Db.Exec(`TRUNCATE TABLE ai_model, website, realm RESTART IDENTITY CASCADE`)
	return err
}

func TestAIModelInsertAndFetch(t *testing.T) {
	// Clean the database before the test
	if err := cleanDatabase(); err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	// Insert a new AIModel
	aiModel := AIModelTest{
		Key:      *NewNullString("test_key"),
		Name:     *NewNullString("Test Model"),
		Type:     *NewNullString("test_type"),
		Provider: *NewNullString("test_provider"),
	}
	err := aiModel.Insert()
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	// Fetch the model by UUID
	fetchedModel, err := AIModelByUUID(aiModel.UUID.String)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	// Compare the inserted and fetched models
	if aiModel.UUID != fetchedModel.UUID {
		t.Errorf("Expected UUID %v, got %v", aiModel.UUID, fetchedModel.UUID)
	}
	if aiModel.Key != fetchedModel.Key {
		t.Errorf("Expected Key %v, got %v", aiModel.Key, fetchedModel.Key)
	}
}

func TestListAIModel(t *testing.T) {
	// Clean the database before the test
	if err := cleanDatabase(); err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	// Insert multiple AIModels
	for i := 1; i <= 50; i++ {
		aiModel := AIModelTest{
			Key:      *NewNullString(fmt.Sprintf("key_%d", i)),
			Name:     *NewNullString(fmt.Sprintf("Model %d", i)),
			Type:     *NewNullString("test_type"),
			Provider: *NewNullString("test_provider"),
		}
		err := aiModel.Insert()
		if err != nil {
			t.Fatalf("Insert error: %v", err)
		}
	}

	// List AIModels with pagination
	perPage := 10
	page := 2
	filters := &Filter{
		"Type": "test_type",
	}
	sort := &Sort{
		"Key": "ASC",
	}
	models, pagination, err := ListAIModel(filters, sort, perPage, page)
	if err != nil {
		t.Fatalf("ListAIModel error: %v", err)
	}

	expectedCount := 50
	if pagination.Count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, pagination.Count)
	}
	expectedPageMax := int(math.Ceil(float64(expectedCount) / float64(perPage)))
	if pagination.PageMax != expectedPageMax {
		t.Errorf("Expected PageMax %d, got %d", expectedPageMax, pagination.PageMax)
	}
	if len(*models) != perPage {
		t.Errorf("Expected %d models, got %d", perPage, len(*models))
	}
}

func TestLinkedFields(t *testing.T) {
	// Clean the database before the test
	if err := cleanDatabase(); err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	// Initialize models
	initRealmModel()
	initWebsiteModel()

	// Insert a new Realm
	realm := RealmTest{
		UUID:      GenNewUUID(""),
		Name:      "Test Realm",
		CreatedAt: NewCustomTime(time.Now()),
		UpdatedAt: NewCustomTime(time.Now()),
	}
	query, args := GetInsertQuery("realm", map[string]interface{}{
		"uuid":       realm.UUID,
		"name":       realm.Name,
		"created_at": realm.CreatedAt,
		"updated_at": realm.UpdatedAt,
	}, "")
	_, err := Db.Exec(query, args...)
	if err != nil {
		t.Fatalf("Failed to insert realm: %v", err)
	}

	// Insert a new Website linked to the Realm
	website := WebsiteTest{
		UUID:      GenNewUUID(""),
		Domain:    "example.com",
		RealmUUID: realm.UUID,
		CreatedAt: *NewCustomTime(time.Now()),
		UpdatedAt: *NewCustomTime(time.Now()),
	}
	query, args = GetInsertQuery("website", map[string]interface{}{
		"uuid":       website.UUID,
		"domain":     website.Domain,
		"realm_uuid": website.RealmUUID,
		"created_at": website.CreatedAt,
		"updated_at": website.UpdatedAt,
	}, "")
	_, err = Db.Exec(query, args...)
	if err != nil {
		t.Fatalf("Failed to insert website: %v", err)
	}

	// Fetch the Website along with the linked Realm
	fetchedWebsite, err := GetWebsiteByUUID(website.UUID)
	if err != nil {
		t.Fatalf("Error fetching website: %v", err)
	}

	// Verify that the linked Realm is correctly fetched
	if fetchedWebsite.Realm == nil {
		t.Fatalf("Expected linked Realm, got nil")
	}
	if fetchedWebsite.Realm.UUID != realm.UUID {
		t.Errorf("Expected Realm UUID %s, got %s", realm.UUID, fetchedWebsite.Realm.UUID)
	}
	if fetchedWebsite.Realm.Name != realm.Name {
		t.Errorf("Expected Realm Name %s, got %s", realm.Name, fetchedWebsite.Realm.Name)
	}
}

func AIModelByUUID(uuidStr string) (*AIModelTest, error) {
	query := aiModelBaseQuery + ` WHERE "ai_model".uuid = $1 LIMIT 1`
	model := AIModelTest{}

	err := Db.Get(&model, query, uuidStr)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (m *AIModelTest) Insert() error {
	query, queryValues := GetInsertQuery("ai_model", map[string]interface{}{
		"uuid":        GenNewUUID(""),
		"key":         m.Key,
		"name":        m.Name,
		"description": m.Description,
		"type":        m.Type,
		"provider":    m.Provider,
		"settings":    m.Settings,
	}, "uuid")

	err := Db.QueryRow(query, queryValues...).Scan(&m.UUID)
	if err != nil {
		return err
	}

	return nil
}

func ListAIModel(filters *Filter, sort *Sort, perPage int, page int) (*[]AIModelTest, *Pagination, error) {
	if sort == nil || len(*sort) == 0 {
		sort = &Sort{
			"Type": "ASC",
		}
	}
	query := aiModelBaseQuery
	query, args, err := FilterQuery(query, "ai_model", filters, sort, "ai_model", perPage, page)
	if err != nil {
		return nil, nil, err
	}

	models := []AIModelTest{}
	err = Db.Select(&models, query, args...)
	if err != nil {
		return nil, nil, err
	}

	countQuery := BuildFilterCount(query)
	count, err := GetFilterCount(countQuery, args)
	if err != nil {
		return nil, nil, err
	}
	pagination := Pagination{
		ResultsPerPage: perPage,
		PageNo:         page,
		Count:          count,
		PageMax:        int(math.Ceil(float64(count) / float64(perPage))),
	}

	return &models, &pagination, nil
}

func GetWebsiteByUUID(uuid string) (*WebsiteTest, error) {
	query := websiteQuerySelectBase + ` WHERE "website".uuid = $1 LIMIT 1`
	website := WebsiteTest{}

	err := Db.Get(&website, query, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &website, nil
}
