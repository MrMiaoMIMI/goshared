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

// NowUnixMilli returns the current Unix timestamp in milliseconds.
func NowUnixMilli() uint64 {
	return uint64(time.Now().UnixMilli())
}

// OperatorProvider resolves the current operator from ctx.
type OperatorProvider func(ctx context.Context) (string, bool)

// TimeProvider returns the timestamp value used by common fields.
//
// The unit is application-defined. The default provider uses Unix milliseconds.
type TimeProvider func() uint64

// CommonFieldOptions configures automatic maintenance for common fields.
type CommonFieldOptions struct {
	// Enabled controls whether executors apply common-field automation at all.
	Enabled bool

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

// DefaultCommonFieldOptions returns the default common-field behavior.
//
// Common-field automation is enabled by default, uses Unix milliseconds for
// ctime/mtime, and resolves creator/updater from ctx with OperatorFromContext.
func DefaultCommonFieldOptions() CommonFieldOptions {
	return CommonFieldOptions{
		Enabled:          true,
		TimeProvider:     NowUnixMilli,
		OperatorProvider: OperatorFromContext,
	}
}

// DisabledCommonFieldOptions disables common-field behavior.
func DisabledCommonFieldOptions() CommonFieldOptions {
	return CommonFieldOptions{}
}

// Normalize fills missing function hooks with defaults.
func (o CommonFieldOptions) Normalize() CommonFieldOptions {
	defaults := DefaultCommonFieldOptions()
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

// DeletedDo provides the standard soft-delete field.
type DeletedDo struct {
	Deleted bool `gorm:"column:deleted;not null;default:false" json:"deleted"`
}

func (c *DeletedDo) GetDeleted() bool {
	if c == nil {
		return false
	}
	return c.Deleted
}

func (c *DeletedDo) SetDeleted(v bool) {
	if c != nil {
		c.Deleted = v
	}
}

func (*DeletedDo) DeletedFieldName() string {
	return DefaultDeletedFieldName
}

// TimeDo provides standard create and update timestamps.
//
// By default, dbhelper-managed executors fill these fields with Unix
// milliseconds. Use dbhelper.WithCommonFieldTimeProvider to customize the unit.
type TimeDo struct {
	Ctime uint64 `gorm:"column:ctime;not null;default:0" json:"ctime"`
	Mtime uint64 `gorm:"column:mtime;not null;default:0" json:"mtime"`
}

func (c *TimeDo) GetCtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Ctime
}

func (c *TimeDo) SetCtime(v uint64) {
	if c != nil {
		c.Ctime = v
	}
}

func (*TimeDo) CtimeFieldName() string {
	return DefaultCtimeFieldName
}

func (c *TimeDo) GetMtime() uint64 {
	if c == nil {
		return 0
	}
	return c.Mtime
}

func (c *TimeDo) SetMtime(v uint64) {
	if c != nil {
		c.Mtime = v
	}
}

func (*TimeDo) MtimeFieldName() string {
	return DefaultMtimeFieldName
}

// OperatorDo provides standard creator and updater fields.
type OperatorDo struct {
	Creator string `gorm:"column:creator;size:255;not null;default:''" json:"creator"`
	Updater string `gorm:"column:updater;size:255;not null;default:''" json:"updater"`
}

func (c *OperatorDo) GetCreator() string {
	if c == nil {
		return ""
	}
	return c.Creator
}

func (c *OperatorDo) SetCreator(v string) {
	if c != nil {
		c.Creator = v
	}
}

func (*OperatorDo) CreatorFieldName() string {
	return DefaultCreatorFieldName
}

func (c *OperatorDo) GetUpdater() string {
	if c == nil {
		return ""
	}
	return c.Updater
}

func (c *OperatorDo) SetUpdater(v string) {
	if c != nil {
		c.Updater = v
	}
}

func (*OperatorDo) UpdaterFieldName() string {
	return DefaultUpdaterFieldName
}

// CommonDo provides the complete standard field set shared by data objects.
type CommonDo struct {
	IdDo
	OperatorDo
	TimeDo
	DeletedDo
}

type IdFieldNamer interface {
	IdFieldName() string
}

type DeletedFieldNamer interface {
	DeletedFieldName() string
}

type IdManaged interface {
	IdFieldNamer
	GetId() uint64
	SetId(uint64)
}

type DeletedManaged interface {
	DeletedFieldNamer
	GetDeleted() bool
	SetDeleted(bool)
}

type CreateTimeManaged interface {
	GetCtime() uint64
	SetCtime(uint64)
	CtimeFieldName() string
}

type UpdateTimeManaged interface {
	GetMtime() uint64
	SetMtime(uint64)
	MtimeFieldName() string
}

type CreatorManaged interface {
	GetCreator() string
	SetCreator(string)
	CreatorFieldName() string
}

type UpdaterManaged interface {
	GetUpdater() string
	SetUpdater(string)
	UpdaterFieldName() string
}

var (
	_ IdManaged         = (*IdDo)(nil)
	_ DeletedManaged    = (*DeletedDo)(nil)
	_ CreateTimeManaged = (*TimeDo)(nil)
	_ UpdateTimeManaged = (*TimeDo)(nil)
	_ CreatorManaged    = (*OperatorDo)(nil)
	_ UpdaterManaged    = (*OperatorDo)(nil)

	_ IdManaged         = (*CommonDo)(nil)
	_ DeletedManaged    = (*CommonDo)(nil)
	_ CreateTimeManaged = (*CommonDo)(nil)
	_ UpdateTimeManaged = (*CommonDo)(nil)
	_ CreatorManaged    = (*CommonDo)(nil)
	_ UpdaterManaged    = (*CommonDo)(nil)
)
