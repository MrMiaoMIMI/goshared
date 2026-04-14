package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Field 通用的日志字段类型，对外屏蔽底层日志库的实现细节
type Field struct {
	zapField zap.Field
}

// toZapField 将 Field 转换为 zap.Field（内部使用）
func (f Field) toZapField() zap.Field {
	return f.zapField
}

// toZapFields 将 []Field 转换为 []zap.Field（内部使用）
func toZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = f.toZapField()
	}
	return zapFields
}

// ============ 基础类型字段构造函数 ============

// String 创建字符串类型字段
func String(key string, val string) Field {
	return Field{zapField: zap.String(key, val)}
}

// Int 创建 int 类型字段
func Int(key string, val int) Field {
	return Field{zapField: zap.Int(key, val)}
}

// Int64 创建 int64 类型字段
func Int64(key string, val int64) Field {
	return Field{zapField: zap.Int64(key, val)}
}

// Int32 创建 int32 类型字段
func Int32(key string, val int32) Field {
	return Field{zapField: zap.Int32(key, val)}
}

// Uint 创建 uint 类型字段
func Uint(key string, val uint) Field {
	return Field{zapField: zap.Uint(key, val)}
}

// Uint64 创建 uint64 类型字段
func Uint64(key string, val uint64) Field {
	return Field{zapField: zap.Uint64(key, val)}
}

// Uint32 创建 uint32 类型字段
func Uint32(key string, val uint32) Field {
	return Field{zapField: zap.Uint32(key, val)}
}

// Float64 创建 float64 类型字段
func Float64(key string, val float64) Field {
	return Field{zapField: zap.Float64(key, val)}
}

// Float32 创建 float32 类型字段
func Float32(key string, val float32) Field {
	return Field{zapField: zap.Float32(key, val)}
}

// Bool 创建 bool 类型字段
func Bool(key string, val bool) Field {
	return Field{zapField: zap.Bool(key, val)}
}

// Duration 创建 time.Duration 类型字段
func Duration(key string, val time.Duration) Field {
	return Field{zapField: zap.Duration(key, val)}
}

// Time 创建 time.Time 类型字段
func Time(key string, val time.Time) Field {
	return Field{zapField: zap.Time(key, val)}
}

// ============ 特殊类型字段构造函数 ============

// Err 创建 error 类型字段，key 固定为 "error"
func Err(err error) Field {
	return Field{zapField: zap.Error(err)}
}

// NamedErr 创建带自定义 key 的 error 类型字段
func NamedErr(key string, err error) Field {
	return Field{zapField: zap.NamedError(key, err)}
}

// Any 创建任意类型字段（使用反射，性能较低，建议优先使用具体类型）
func Any(key string, val interface{}) Field {
	return Field{zapField: zap.Any(key, val)}
}

// Stringer 创建 fmt.Stringer 类型字段
func Stringer(key string, val fmt.Stringer) Field {
	return Field{zapField: zap.Stringer(key, val)}
}

// Binary 创建二进制数据字段（base64 编码输出）
func Binary(key string, val []byte) Field {
	return Field{zapField: zap.Binary(key, val)}
}

// Stack 创建调用栈字段（当前 goroutine 的完整栈跟踪）
func Stack(key string) Field {
	return Field{zapField: zap.Stack(key)}
}

// Namespace 创建命名空间字段，后续字段会嵌套在此命名空间下
func Namespace(key string) Field {
	return Field{zapField: zap.Namespace(key)}
}

// ============ 数组/切片类型字段构造函数 ============

// Strings 创建字符串切片类型字段
func Strings(key string, val []string) Field {
	return Field{zapField: zap.Strings(key, val)}
}

// Ints 创建 int 切片类型字段
func Ints(key string, val []int) Field {
	return Field{zapField: zap.Ints(key, val)}
}

// Int64s 创建 int64 切片类型字段
func Int64s(key string, val []int64) Field {
	return Field{zapField: zap.Int64s(key, val)}
}

// Uint64s 创建 uint64 切片类型字段
func Uint64s(key string, val []uint64) Field {
	return Field{zapField: zap.Uint64s(key, val)}
}

// Float64s 创建 float64 切片类型字段
func Float64s(key string, val []float64) Field {
	return Field{zapField: zap.Float64s(key, val)}
}

// Bools 创建 bool 切片类型字段
func Bools(key string, val []bool) Field {
	return Field{zapField: zap.Bools(key, val)}
}

