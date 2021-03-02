package model

type FileInfo struct {
	FileId string `json:"file_id"`
	Name   string `json:"name"`
	ReName string `json:"rename"`
	Url    string `json:"url"`
	Path   string `json:"path"`
	Md5    string `json:"md5"`
	Size   int64  `json:"size"`
	Group  string `json:"group"`
}

type SyncFileInfo struct {
	Src      string `json:"src"`
	Dst      string `json:"dst"`
	FileId   string `json:"file_id"`
	FilePath string `json:"file_path"`
	FileName string `json:"file_name"`
	FileHash string `json:"file_hash"`
	Action   string `json:"action"`
	Group    string `json:"group"`
}
