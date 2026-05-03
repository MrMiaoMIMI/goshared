package dbspi

import (
	"context"
	"time"
)

type operatorCtxKey struct{}

// WithOperator injects the current operator into ctx for common-field autofill.
func WithOperator(ctx context.Context, operator string) context.Context {
	return context.WithValue(ctx, operatorCtxKey{}, operator)
}

// OperatorFromContext extracts the current operator from ctx.
func OperatorFromContext(ctx context.Context) (string, bool) {
	operator, ok := ctx.Value(operatorCtxKey{}).(string)
	return operator, ok
}

// DefaultTimeProvider returns the current Unix timestamp in milliseconds.
func DefaultTimeProvider() uint64 {
	return uint64(time.Now().UnixMilli())
}

// OperatorProvider resolves the current operator from ctx.
type OperatorProvider func(ctx context.Context) (string, bool)

// TimeProvider returns the timestamp value used by common fields.
//
// The unit is application-defined. The default provider uses Unix milliseconds.
type TimeProvider func() uint64

// CommonFieldAutoFillOptions configures automatic maintenance for common fields.
type CommonFieldAutoFillOptions struct {
	// AutoFillEnabled controls whether executors apply common-field automation at all.
	AutoFillEnabled bool

	// OverwriteExplicitValues controls whether generated common-field values may
	// overwrite values already provided by the caller.
	//
	// When false, common-field automation only fills zero-value ctime/mtime and
	// empty creator/updater. UpdateByQuery also keeps explicit mtime/updater
	// values in the updater. Set this to true to force generated values to
	// replace explicit values.
	OverwriteExplicitValues bool

	// TimeProvider generates values for ctime and mtime. If nil, Normalize uses
	// the default Unix-millisecond provider.
	TimeProvider TimeProvider

	// OperatorProvider resolves creator and updater values from ctx. If nil,
	// Normalize uses OperatorFromContext.
	OperatorProvider OperatorProvider
}

// DefaultCommonFieldAutoFillOptions returns the default common-field behavior.
//
// Common-field automation is enabled by default, uses Unix milliseconds for
// ctime/mtime, and resolves creator/updater from ctx with OperatorFromContext.
func DefaultCommonFieldAutoFillOptions() CommonFieldAutoFillOptions {
	return CommonFieldAutoFillOptions{
		AutoFillEnabled:  true,
		TimeProvider:     DefaultTimeProvider,
		OperatorProvider: OperatorFromContext,
	}
}

// DisabledCommonFieldAutoFillOptions disables common-field behavior.
func DisabledCommonFieldAutoFillOptions() CommonFieldAutoFillOptions {
	return CommonFieldAutoFillOptions{}
}

// Normalize fills missing function hooks with defaults.
func (o CommonFieldAutoFillOptions) Normalize() CommonFieldAutoFillOptions {
	defaults := DefaultCommonFieldAutoFillOptions()
	if o.TimeProvider == nil {
		o.TimeProvider = defaults.TimeProvider
	}
	if o.OperatorProvider == nil {
		o.OperatorProvider = defaults.OperatorProvider
	}
	return o
}

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
