package dbspi

// IdDo provides the standard primary id field.
type IdDo struct {
	Id uint64 `gorm:"primaryKey;column:id" json:"id"`
}

func (c *IdDo) GetId() uint64 {
	if c == nil {
		return 0
	}
	return c.Id
}

func (c *IdDo) SetId(v uint64) {
	if c != nil {
		c.Id = v
	}
}

func (*IdDo) IdFieldName() string {
	return DefaultIdFieldName
}

// SoftDeleteDo provides the standard soft-delete field.
type SoftDeleteDo struct {
	Deleted bool `gorm:"column:deleted;not null;default:false" json:"deleted"`
}

func (c *SoftDeleteDo) GetDeleted() bool {
	if c == nil {
		return false
	}
	return c.Deleted
}

func (c *SoftDeleteDo) SetDeleted(v bool) {
	if c != nil {
		c.Deleted = v
	}
}

func (*SoftDeleteDo) DeletedFieldName() string {
	return DefaultDeletedFieldName
}

// CreateTimeDo provides the standard create timestamp field.
//
// By default, dbhelper-managed executors fill this field with Unix
// milliseconds. Use dbhelper.WithCommonFieldTimeProvider to customize the unit.
type CreateTimeDo struct {
	Ctime uint64 `gorm:"column:ctime;not null;default:0" json:"ctime"`
}

func (c *CreateTimeDo) GetCtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Ctime
}

func (c *CreateTimeDo) SetCtime(v uint64) {
	if c != nil {
		c.Ctime = v
	}
}

func (*CreateTimeDo) CtimeFieldName() string {
	return DefaultCtimeFieldName
}

// UpdateTimeDo provides the standard update timestamp field.
//
// By default, dbhelper-managed executors fill this field with Unix
// milliseconds. Use dbhelper.WithCommonFieldTimeProvider to customize the unit.
type UpdateTimeDo struct {
	Mtime uint64 `gorm:"column:mtime;not null;default:0" json:"mtime"`
}

func (c *UpdateTimeDo) GetMtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Mtime
}

func (c *UpdateTimeDo) SetMtime(v uint64) {
	if c != nil {
		c.Mtime = v
	}
}

func (*UpdateTimeDo) MtimeFieldName() string {
	return DefaultMtimeFieldName
}

// TimeDo provides standard create and update timestamp fields.
type TimeDo struct {
	CreateTimeDo
	UpdateTimeDo
}

// CreatorDo provides the standard creator field.
type CreatorDo struct {
	Creator string `gorm:"column:creator;size:255;not null;default:''" json:"creator"`
}

func (c *CreatorDo) GetCreator() string {
	if c == nil {
		return ""
	}
	return c.Creator
}

func (c *CreatorDo) SetCreator(v string) {
	if c != nil {
		c.Creator = v
	}
}

func (*CreatorDo) CreatorFieldName() string {
	return DefaultCreatorFieldName
}

// UpdaterDo provides the standard updater field.
type UpdaterDo struct {
	Updater string `gorm:"column:updater;size:255;not null;default:''" json:"updater"`
}

func (c *UpdaterDo) GetUpdater() string {
	if c == nil {
		return ""
	}
	return c.Updater
}

func (c *UpdaterDo) SetUpdater(v string) {
	if c != nil {
		c.Updater = v
	}
}

func (*UpdaterDo) UpdaterFieldName() string {
	return DefaultUpdaterFieldName
}

// OperatorDo provides standard creator and updater fields.
type OperatorDo struct {
	CreatorDo
	UpdaterDo
}

// CommonDo provides the complete standard field set shared by data objects.
type CommonDo struct {
	IdDo
	CreatorDo
	UpdaterDo
	CreateTimeDo
	UpdateTimeDo
	SoftDeleteDo
}
