package dbspi

// Manager is an opaque database manager handle returned by dbhelper.NewManager.
//
// Do not implement this interface directly. dbhelper only supports Manager
// values returned by dbhelper.NewManager or dbhelper.DefaultManager.
type Manager interface {
	ManagerHandle()
}
