// Package mqspi defines the public MQ contracts and configuration types.
//
// Business code should depend on this package for interface types, message
// shapes, configs, credentials, and failure policies. Config structs are meant
// to be built directly or loaded from configuration files. Runtime factories
// and message convenience helpers live in mqhelper.
package mqspi
