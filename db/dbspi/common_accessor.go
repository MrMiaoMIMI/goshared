package dbspi

// IdFieldNameProvider customizes the id column name used by id-based helpers.
type IdFieldNameProvider interface {
	IdFieldName() string
}

// SoftDeleteFieldNameProvider customizes the soft-delete column name.
type SoftDeleteFieldNameProvider interface {
	SoftDeleteFieldName() string
}

// IdAccessor reads and writes the standard id field.
type IdAccessor interface {
	IdFieldNameProvider
	GetId() uint64
	SetId(uint64)
}

// SoftDeleteAccessor reads and writes the standard soft-delete field.
type SoftDeleteAccessor interface {
	SoftDeleteFieldNameProvider
	GetDeleted() bool
	SetDeleted(bool)
}

// CreateTimeAccessor reads and writes the standard create timestamp field.
type CreateTimeAccessor interface {
	GetCtime() uint64
	SetCtime(uint64)
	CtimeFieldName() string
}

// UpdateTimeAccessor reads and writes the standard update timestamp field.
type UpdateTimeAccessor interface {
	GetMtime() uint64
	SetMtime(uint64)
	MtimeFieldName() string
}

// CreatorAccessor reads and writes the standard creator field.
type CreatorAccessor interface {
	GetCreator() string
	SetCreator(string)
	CreatorFieldName() string
}

// UpdaterAccessor reads and writes the standard updater field.
type UpdaterAccessor interface {
	GetUpdater() string
	SetUpdater(string)
	UpdaterFieldName() string
}

var (
	_ IdAccessor         = (*IdField)(nil)
	_ SoftDeleteAccessor = (*SoftDeleteField)(nil)
	_ CreateTimeAccessor = (*CreateTimeField)(nil)
	_ UpdateTimeAccessor = (*UpdateTimeField)(nil)
	_ CreatorAccessor    = (*CreatorField)(nil)
	_ UpdaterAccessor    = (*UpdaterField)(nil)
	_ CreateTimeAccessor = (*TimeFields)(nil)
	_ UpdateTimeAccessor = (*TimeFields)(nil)
	_ CreatorAccessor    = (*OperatorFields)(nil)
	_ UpdaterAccessor    = (*OperatorFields)(nil)

	_ IdAccessor         = (*CommonFields)(nil)
	_ SoftDeleteAccessor = (*CommonFields)(nil)
	_ CreateTimeAccessor = (*CommonFields)(nil)
	_ UpdateTimeAccessor = (*CommonFields)(nil)
	_ CreatorAccessor    = (*CommonFields)(nil)
	_ UpdaterAccessor    = (*CommonFields)(nil)
)
