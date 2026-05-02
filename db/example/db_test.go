package example

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

const (
	testDbHost     = "127.0.0.1"
	testDbPort     = 3306
	testDbUser     = "root"
	testDbPassword = "123456"
	testDbName     = "my_test"
	testAppDbName  = "my_app_db"
	testDebugMode  = true
)

// Example demonstrates how to use the GORM implementation of dbspi

// User model example
type User struct {
	ID      int64  `gorm:"primaryKey"`
	Name    string `gorm:"column:name"`
	Email   string `gorm:"column:email"`
	Age     int    `gorm:"column:age"`
	Status  string `gorm:"column:status"`
	Deleted bool   `gorm:"column:deleted"`
}

func (*User) TableName() string {
	return "dbspi_test_user_tab"
}

func (*User) IdFieldName() string {
	return dbspi.DefaultIdFieldName
}

// UserTable represents the user table with type-safe fields
type UserFieldManager struct {
	ID      dbspi.Field[int64]
	Name    dbspi.Field[string]
	Email   dbspi.Field[string]
	Age     dbspi.Field[int]
	Status  dbspi.Field[string]
	Deleted dbspi.Field[bool]
}

// NewUserFieldManager creates a new UserFieldManager with field definitions
func NewUserFieldManager() *UserFieldManager {
	return &UserFieldManager{
		ID:      dbhelper.NewField[int64](dbspi.DefaultIdFieldName),
		Name:    dbhelper.NewField[string]("name"),
		Email:   dbhelper.NewField[string]("email"),
		Age:     dbhelper.NewField[int]("age"),
		Status:  dbhelper.NewField[string]("status"),
		Deleted: dbhelper.NewField[bool](dbspi.DefaultDeletedFieldName),
	}
}

func testDbManager(dbName string) dbspi.DbManager {
	return dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host:     testDbHost,
				Port:     testDbPort,
				User:     testDbUser,
				Password: testDbPassword,
				DbName:   dbName,
				Debug:    testDebugMode,
			},
		},
	})
}

func testDSN(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		testDbUser, testDbPassword, testDbHost, testDbPort, dbName)
}

// Helper functions for pointer creation
func ptrInt(i int) *int {
	return &i
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
