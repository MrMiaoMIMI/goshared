package dbspi

type IdFieldNamer interface {
	IdFieldName() string
}

type SoftDeleteFieldNamer interface {
	DeletedFieldName() string
}

type IdAccessor interface {
	IdFieldNamer
	GetId() uint64
	SetId(uint64)
}

type SoftDeleteAccessor interface {
	SoftDeleteFieldNamer
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
	_ IdAccessor         = (*IdDo)(nil)
	_ SoftDeleteAccessor = (*SoftDeleteDo)(nil)
	_ CreateTimeAccessor = (*CreateTimeDo)(nil)
	_ UpdateTimeAccessor = (*UpdateTimeDo)(nil)
	_ CreatorAccessor    = (*CreatorDo)(nil)
	_ UpdaterAccessor    = (*UpdaterDo)(nil)
	_ CreateTimeAccessor = (*TimeDo)(nil)
	_ UpdateTimeAccessor = (*TimeDo)(nil)
	_ CreatorAccessor    = (*OperatorDo)(nil)
	_ UpdaterAccessor    = (*OperatorDo)(nil)

	_ IdAccessor         = (*CommonDo)(nil)
	_ SoftDeleteAccessor = (*CommonDo)(nil)
	_ CreateTimeAccessor = (*CommonDo)(nil)
	_ UpdateTimeAccessor = (*CommonDo)(nil)
	_ CreatorAccessor    = (*CommonDo)(nil)
	_ UpdaterAccessor    = (*CommonDo)(nil)
)
