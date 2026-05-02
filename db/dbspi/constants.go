package dbspi

const (
	// DefaultDbKey is the database config key used when an entity does not
	// implement DbKeyProvider.
	DefaultDbKey = "default"

	// Default common field names used by CommonDo and executor helper methods.
	DefaultIdFieldName      = "id"
	DefaultDeletedFieldName = "deleted"
	DefaultCreatorFieldName = "creator"
	DefaultUpdaterFieldName = "updater"
	DefaultCtimeFieldName   = "ctime"
	DefaultMtimeFieldName   = "mtime"

	// Default connection pool settings applied when DbServerConfig leaves the
	// corresponding field as zero.
	DefaultMaxOpenConns           = 100
	DefaultMaxIdleConns           = 10
	DefaultConnMaxLifetimeSeconds = 3600
)
