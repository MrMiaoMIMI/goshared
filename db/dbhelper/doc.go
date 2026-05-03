// Package dbhelper provides user-facing factory and builder helpers for the db
// module.
//
// Use this package to create dbspi.Manager handles, bind an entity type to a
// table store, bind a table store to a transaction with WithTx, and build
// query/update/pagination values backed by the internal implementation. Public
// contracts, configs, and model composition types live in package dbspi.
package dbhelper
