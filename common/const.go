package common

const VERSION = "1.0.0"

//部署方式
const (
	DeployTypeStorages = "storage"
	DeployTypeTracker  = "tracker"
)

//返回码
const (
	Success = 20000 + iota

	DirCreateFail = 40000 + iota
	FormFileNotFound
	FileSizeExceeded
	FileSaveFail
)

//http请求头
const (
	HeaderUploadFileDir = "Egg-Dfs-FileDir"
	HeaderUUIDFileName  = "Egg-Dfs-FileName"
)

//group状态标识
const (
	GroupUnavailable = iota
	GroupActive
)

//storage状态标识
const (
	StorageOffline = iota
	StorageActive
	StorageNotEnoughSpace
)
