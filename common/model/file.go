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
