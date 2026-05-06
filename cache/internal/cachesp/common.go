package cachesp

import (
	"fmt"
	"reflect"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
)

type valueCodec interface {
	encode(value any) (any, error)
	decode(value any, receiver any) error
}

type referenceValueCodec struct{}

func (referenceValueCodec) encode(value any) (any, error) {
	return value, nil
}

func (referenceValueCodec) decode(value any, receiver any) error {
	return setReceiver(receiver, value)
}

type binaryValueCodec struct {
	codec cachespi.Codec
}

func newBinaryValueCodec(codec cachespi.Codec) binaryValueCodec {
	if codec == nil {
		codec = cachespi.JSONCodec{}
	}
	return binaryValueCodec{codec: codec}
}

func (c binaryValueCodec) encode(value any) (any, error) {
	return c.encodeBytes(value)
}

func (c binaryValueCodec) encodeBytes(value any) ([]byte, error) {
	return c.codec.Marshal(value)
}

func (c binaryValueCodec) decode(value any, receiver any) error {
	data, err := bytesFromStoredValue(value)
	if err != nil {
		return err
	}
	return c.decodeBytes(data, receiver)
}

func (c binaryValueCodec) decodeBytes(data []byte, receiver any) error {
	return c.codec.Unmarshal(data, receiver)
}

func bytesFromStoredValue(value any) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("cache: expected serialized bytes, got %T", value)
	}
}

func resolveTTL(expire time.Duration, defaultTTL time.Duration) (time.Duration, error) {
	switch expire {
	case cachespi.NoExpiration:
		return 0, nil
	case cachespi.DefaultExpiration:
		return normalizeTTL(defaultTTL)
	default:
		return normalizeTTL(expire)
	}
}

func normalizeTTL(ttl time.Duration) (time.Duration, error) {
	if ttl == cachespi.NoExpiration {
		return 0, nil
	}
	if ttl < 0 {
		return 0, fmt.Errorf("%w: %s", cachespi.ErrInvalidExpiration, ttl)
	}
	return ttl, nil
}

func setReceiver(receiver any, value any) error {
	if receiver == nil {
		return fmt.Errorf("cache: receiver must be a non-nil pointer")
	}
	rv := reflect.ValueOf(receiver)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("cache: receiver must be a non-nil pointer")
	}

	target := rv.Elem()
	targetType := target.Type()
	if value == nil {
		target.Set(reflect.Zero(targetType))
		return nil
	}

	val := reflect.ValueOf(value)
	if val.Type().AssignableTo(targetType) {
		target.Set(val)
		return nil
	}

	if val.Kind() == reflect.Ptr && !val.IsNil() && val.Elem().Type().AssignableTo(targetType) {
		target.Set(val.Elem())
		return nil
	}

	if targetType.Kind() == reflect.Ptr && val.Type().AssignableTo(targetType.Elem()) {
		ptr := reflect.New(targetType.Elem())
		ptr.Elem().Set(val)
		target.Set(ptr)
		return nil
	}

	return fmt.Errorf("cache: cannot assign %T to %s", value, targetType)
}
