package common

const (
	DeployTypeStorages = "storage"
	DeployTypeTracker  = "tracker"
)

const (
	Success = 20000 + iota

	DirCreateFail = 40000 + iota
	FormFileNotFound
	FileSizeExceeded
	FileSaveFail
)
