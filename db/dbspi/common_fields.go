package dbspi

// IdField provides the standard primary id field.
type IdField struct {
	Id uint64 `gorm:"primaryKey;column:id" json:"id"`
}

func (c *IdField) GetId() uint64 {
	if c == nil {
		return 0
	}
	return c.Id
}

func (c *IdField) SetId(v uint64) {
	if c != nil {
		c.Id = v
	}
}

func (*IdField) IdFieldName() string {
	return DefaultIdFieldName
}

// SoftDeleteField provides the standard soft-delete field.
type SoftDeleteField struct {
	Deleted bool `gorm:"column:deleted;not null;default:false" json:"deleted"`
}

func (c *SoftDeleteField) GetDeleted() bool {
	if c == nil {
		return false
	}
	return c.Deleted
}

func (c *SoftDeleteField) SetDeleted(v bool) {
	if c != nil {
		c.Deleted = v
	}
}

func (*SoftDeleteField) SoftDeleteFieldName() string {
	return DefaultDeletedFieldName
}

// CreateTimeField provides the standard create timestamp field.
//
// By default, dbhelper-managed executors fill this field with Unix
// milliseconds. Use dbhelper.WithCommonFieldTimeProvider to customize the unit.
type CreateTimeField struct {
	Ctime uint64 `gorm:"column:ctime;not null;default:0" json:"ctime"`
}

func (c *CreateTimeField) GetCtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Ctime
}

func (c *CreateTimeField) SetCtime(v uint64) {
	if c != nil {
		c.Ctime = v
	}
}

func (*CreateTimeField) CtimeFieldName() string {
	return DefaultCtimeFieldName
}

// UpdateTimeField provides the standard update timestamp field.
//
// By default, dbhelper-managed executors fill this field with Unix
// milliseconds. Use dbhelper.WithCommonFieldTimeProvider to customize the unit.
type UpdateTimeField struct {
	Mtime uint64 `gorm:"column:mtime;not null;default:0" json:"mtime"`
}

func (c *UpdateTimeField) GetMtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Mtime
}

func (c *UpdateTimeField) SetMtime(v uint64) {
	if c != nil {
		c.Mtime = v
	}
}

func (*UpdateTimeField) MtimeFieldName() string {
	return DefaultMtimeFieldName
}

// TimeFields provides standard create and update timestamp fields.
type TimeFields struct {
	CreateTimeField
	UpdateTimeField
}

// CreatorField provides the standard creator field.
type CreatorField struct {
	Creator string `gorm:"column:creator;size:255;not null;default:''" json:"creator"`
}

func (c *CreatorField) GetCreator() string {
	if c == nil {
		return ""
	}
	return c.Creator
}

func (c *CreatorField) SetCreator(v string) {
	if c != nil {
		c.Creator = v
	}
}

func (*CreatorField) CreatorFieldName() string {
	return DefaultCreatorFieldName
}

// UpdaterField provides the standard updater field.
type UpdaterField struct {
	Updater string `gorm:"column:updater;size:255;not null;default:''" json:"updater"`
}

func (c *UpdaterField) GetUpdater() string {
	if c == nil {
		return ""
	}
	return c.Updater
}

func (c *UpdaterField) SetUpdater(v string) {
	if c != nil {
		c.Updater = v
	}
}

func (*UpdaterField) UpdaterFieldName() string {
	return DefaultUpdaterFieldName
}

// OperatorFields provides standard creator and updater fields.
type OperatorFields struct {
	CreatorField
	UpdaterField
}

// CommonFields provides the complete standard field set shared by data objects.
type CommonFields struct {
	IdField
	CreatorField
	UpdaterField
	CreateTimeField
	UpdateTimeField
	SoftDeleteField
}
