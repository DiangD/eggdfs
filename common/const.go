package common

const VERSION = "1.0.0"

//部署方式
const (
	DeployTypeStorages = "storage"
	DeployTypeTracker  = "tracker"
)

//返回码
const (
	//common
	Success = 20000 + iota

	Fail = 10000
)

//error code
const (
	DirCreateFail = 40000 + iota
	FormFileNotFound
	FileSizeExceeded
	FileSaveFail
	ProxyBadGateWay
	FileDamaged
)

//http请求头
const (
	HeaderUploadFileDir = "Egg-Dfs-FileDir"
	HeaderFileUUID      = "Egg-Dfs-FileUUID"
	HeaderFileUploadRes = "Egg-Dfs-Upload-Res"
	HeaderFileHash      = "Egg-Dfs-FIle-Hash"
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

const (
	SyncAdd    = "ADD"
	SyncDelete = "DELETE"
)
