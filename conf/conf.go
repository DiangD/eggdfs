package conf

import (
	"github.com/spf13/viper"
	"sync/atomic"
	"unsafe"
)

const (
	DefaultConfigFileName = "eggdfs_config"
)

var configPtr unsafe.Pointer

type GlobalConfig struct {
	DeployType string `json:"deploy_type"`
	Port       string `json:"port"`
	Host       string `json:"host"`
	LogDir     string `json:"log_dir"`
	Group      string `json:"group"`
	StorageDir string `json:"storage_dir"`

	Tracker struct {
		NodeId        string `json:"node_id"`
		EnableTmpFile bool   `json:"enable_tmp_file"`
	} `json:"tracker"`

	Storage struct {
		Group         string   `json:"group"`
		FileSizeLimit int64    `json:"file_size_limit"`
		StorageDir    string   `json:"storage_dir"`
		Trackers      []string `json:"trackers"`
	} `json:"storage"`
}

func ParseConfig() {
	v := viper.New()
	v.AddConfigPath("..")
	v.AddConfigPath("../config")
	v.SetConfigName(DefaultConfigFileName)
	v.SetConfigType("json")

	err := v.ReadInConfig()
	if err != nil {
		panic("config not exist")
	}
	c := GlobalConfig{}
	err = v.Unmarshal(&c)
	if err != nil {
		panic("parse config file error,please check config")
	}
	//并发安全，指针赋值
	atomic.StorePointer(&configPtr, unsafe.Pointer(&c))
}

func Config() *GlobalConfig {
	return (*GlobalConfig)(atomic.LoadPointer(&configPtr))
}
