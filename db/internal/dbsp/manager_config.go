package dbsp

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

func validateDatabaseTargetsMatchRule(dbs []DatabaseTarget, rule DatabaseShardingRule) error {
	enumerator, ok := rule.(interface {
		EnumerateDbNames() ([]string, error)
	})
	if !ok {
		return nil
	}

	expectedKeys, err := enumerator.EnumerateDbNames()
	if err != nil {
		return fmt.Errorf("enumerate database sharding targets: %w", err)
	}
	actualKeys := make(map[string]struct{}, len(dbs))
	for _, db := range dbs {
		actualKeys[db.Key] = struct{}{}
	}
	for _, key := range expectedKeys {
		if _, ok := actualKeys[key]; !ok {
			return fmt.Errorf("database_sharding resolves target %q, but no matching server key exists", key)
		}
	}
	return nil
}

func validateDatabaseGroupConfig(entry dbspi.DatabaseGroupConfig) error {
	if entry.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrency must be >= 0")
	}
	if err := validateConnectionPool(entry.MaxOpenConns, entry.MaxIdleConns, entry.ConnMaxLifetimeSeconds); err != nil {
		return err
	}
	if entry.DatabaseSharding != nil {
		if entry.DatabaseSharding.NameExpr == "" {
			return fmt.Errorf("database_sharding.name_expr is required")
		}
		if len(entry.Servers) == 0 && entry.DSN != "" {
			return fmt.Errorf("DSN cannot be used with database_sharding on a single server " +
				"(DSN includes the database name). Use Host/Port/User/Password fields instead, " +
				"or use the Servers list with per-server DSN")
		}
	}
	if entry.TableSharding != nil && entry.TableSharding.NameExpr == "" {
		return fmt.Errorf("table_sharding.name_expr is required")
	}

	if len(entry.Servers) > 0 {
		seenKeys := make(map[string]struct{}, len(entry.Servers))
		for _, server := range entry.Servers {
			if server.Key == "" {
				return fmt.Errorf("servers[].key is required")
			}
			if _, ok := seenKeys[server.Key]; ok {
				return fmt.Errorf("duplicate server key %q", server.Key)
			}
			seenKeys[server.Key] = struct{}{}
			if err := validateServerConfig(server.ServerConfig, true); err != nil {
				return fmt.Errorf("server %q: %w", server.Key, err)
			}
		}
	} else {
		requireDatabaseName := entry.DatabaseSharding == nil
		if err := validateServerConfig(toServerConfig(entry), requireDatabaseName); err != nil {
			return err
		}
	}

	seenTables := make(map[string]struct{})
	for _, rule := range entry.TableRules {
		if len(rule.Tables) == 0 {
			return fmt.Errorf("table_rules[].tables is required")
		}
		if rule.MaxConcurrency != nil && *rule.MaxConcurrency < 0 {
			return fmt.Errorf("table_rules for %v: max_concurrency must be >= 0", rule.Tables)
		}
		if rule.TableSharding != nil && rule.TableSharding.NameExpr == "" && entry.TableSharding == nil {
			return fmt.Errorf("table_rules for %v: table_sharding.name_expr is required", rule.Tables)
		}
		for _, tableName := range rule.Tables {
			if tableName == "" {
				return fmt.Errorf("table_rules[].tables contains an empty table name")
			}
			if _, ok := seenTables[tableName]; ok {
				return fmt.Errorf("duplicate table rule for table %q", tableName)
			}
			seenTables[tableName] = struct{}{}
		}
	}

	return nil
}

func validateServerConfig(cfg dbspi.ServerConfig, requireDatabaseName bool) error {
	if err := validateConnectionPool(cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetimeSeconds); err != nil {
		return err
	}
	if cfg.DSN != "" {
		return nil
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required when dsn is empty")
	}
	if cfg.Port == 0 {
		return fmt.Errorf("port is required when dsn is empty")
	}
	if cfg.User == "" {
		return fmt.Errorf("user is required when dsn is empty")
	}
	if requireDatabaseName && cfg.DatabaseName == "" {
		return fmt.Errorf("database_name is required when dsn is empty")
	}
	return nil
}

func validateConnectionPool(maxOpenConns int, maxIdleConns int, connMaxLifetimeSeconds int) error {
	if maxOpenConns < 0 {
		return fmt.Errorf("max_open_conns must be >= 0")
	}
	if maxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns must be >= 0")
	}
	if connMaxLifetimeSeconds < 0 {
		return fmt.Errorf("conn_max_lifetime_seconds must be >= 0")
	}
	return nil
}
