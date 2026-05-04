package dbhelper

import (
	"strings"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

func TestNewManagerAcceptsDocumentedDatabaseGroupConfigModes(t *testing.T) {
	baseServer := dbspi.DatabaseGroupConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "123456",
	}
	tableSharding := &dbspi.TableShardingConfig{
		NameExpr: "order_tab_${index}",
		ExpandExprs: []string{
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		},
	}
	databaseSharding := &dbspi.DatabaseShardingConfig{
		NameExpr: "order_db_${idx}",
		ExpandExprs: []string{
			"${idx} := range(0, 2)",
			"${idx} = @{shop_id} % 2",
		},
	}

	tests := []struct {
		name  string
		group dbspi.DatabaseGroupConfig
	}{
		{
			name: "single database",
			group: dbspi.DatabaseGroupConfig{
				DSN: "root:123456@tcp(127.0.0.1:3306)/my_app_db?charset=utf8mb4&parseTime=True&loc=Local",
			},
		},
		{
			name: "single server database sharding",
			group: func() dbspi.DatabaseGroupConfig {
				group := baseServer
				group.DatabaseSharding = databaseSharding
				return group
			}(),
		},
		{
			name: "multi server database sharding",
			group: dbspi.DatabaseGroupConfig{
				Servers: []dbspi.NamedServerConfig{
					{
						Key: "order_db_0",
						ServerConfig: dbspi.ServerConfig{
							DSN: "root:123456@tcp(127.0.0.1:3306)/order_db_0?charset=utf8mb4&parseTime=True&loc=Local",
						},
					},
					{
						Key: "order_db_1",
						ServerConfig: dbspi.ServerConfig{
							DSN: "root:123456@tcp(127.0.0.1:3306)/order_db_1?charset=utf8mb4&parseTime=True&loc=Local",
						},
					},
				},
				DatabaseSharding: databaseSharding,
			},
		},
		{
			name: "table sharding only",
			group: func() dbspi.DatabaseGroupConfig {
				group := baseServer
				group.DatabaseName = "my_app_db"
				group.TableSharding = tableSharding
				return group
			}(),
		},
		{
			name: "database and table sharding",
			group: func() dbspi.DatabaseGroupConfig {
				group := baseServer
				group.DatabaseSharding = databaseSharding
				group.TableSharding = tableSharding
				return group
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(dbspi.DatabaseConfig{
				DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
					dbspi.DefaultDatabaseGroupKey: tt.group,
				},
			})
			if err != nil {
				t.Fatalf("NewManager error = %v", err)
			}
		})
	}
}

func TestNewManagerRejectsEmptyConfig(t *testing.T) {
	_, err := NewManager(dbspi.DatabaseConfig{})
	if err == nil || !strings.Contains(err.Error(), "DatabaseGroups is empty") {
		t.Fatalf("NewManager error = %v, want empty DatabaseGroups error", err)
	}
}

func TestNewManagerRejectsSingleServerDatabaseShardingWithDSN(t *testing.T) {
	_, err := NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				DSN: "root:123456@tcp(127.0.0.1:3306)/my_test?charset=utf8mb4&parseTime=True&loc=Local",
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "DSN cannot be used with database_sharding") {
		t.Fatalf("NewManager error = %v, want DSN/database_sharding error", err)
	}
}

func TestNewManagerWrapsShardingRuleConfigError(t *testing.T) {
	_, err := NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			"order_dbs": {
				Host: "127.0.0.1", Port: 3306, User: "root",
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_db_${idx",
					ExpandExprs: []string{"${idx} := range(0, 2)"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected NewManager error")
	}
	if !strings.Contains(err.Error(), `database group "order_dbs"`) ||
		!strings.Contains(err.Error(), "parse db name_expr") {
		t.Fatalf("NewManager error = %v, want group key and sharding parse error", err)
	}
}

func TestNewManagerRejectsDatabaseShardingTargetMismatch(t *testing.T) {
	_, err := NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			"order_dbs": {
				Servers: []dbspi.NamedServerConfig{
					{
						Key: "SG",
						ServerConfig: dbspi.ServerConfig{
							DSN: "root:123456@tcp(127.0.0.1:3306)/order_SG_db?charset=utf8mb4&parseTime=True&loc=Local",
						},
					},
				},
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_${region}_db",
					ExpandExprs: []string{"${region} := enum(SG)"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected NewManager error")
	}
	if !strings.Contains(err.Error(), `database group "order_dbs"`) ||
		!strings.Contains(err.Error(), `no matching server key`) {
		t.Fatalf("NewManager error = %v, want target mismatch error", err)
	}
}
