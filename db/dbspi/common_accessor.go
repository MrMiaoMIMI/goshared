package dbspi

type IdFieldNameProvider interface {
	IdFieldName() string
}

type SoftDeleteFieldNameProvider interface {
	SoftDeleteFieldName() string
}

type IdAccessor interface {
	IdFieldNameProvider
	GetId() uint64
	SetId(uint64)
}

type SoftDeleteAccessor interface {
	SoftDeleteFieldNameProvider
	GetDeleted() bool
	SetDeleted(bool)
}

type CreateTimeAccessor interface {
	GetCtime() uint64
	SetCtime(uint64)
	CtimeFieldName() string
}

type UpdateTimeAccessor interface {
	GetMtime() uint64
	SetMtime(uint64)
	MtimeFieldName() string
}

type CreatorAccessor interface {
	GetCreator() string
	SetCreator(string)
	CreatorFieldName() string
}

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
