package common

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
