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
	DeployType string `mapstructure:"deploy_type"`
	Port       string `mapstructure:"port"`
	Host       string `mapstructure:"host"`
	LogDir     string `mapstructure:"log_dir"`
	StorageDir string `mapstructure:"storage_dir"`

	Tracker struct {
		NodeId        string `mapstructure:"node_id"`
		EnableTmpFile bool   `mapstructure:"enable_tmp_file"`
	} `json:"tracker"`

	Storage struct {
		Group         string   `mapstructure:"group"`
		FileSizeLimit int64    `mapstructure:"file_size_limit"`
		StorageDir    string   `mapstructure:"storage_dir"`
		Trackers      []string `mapstructure:"trackers"`
	} `json:"storage"`
}

func parseConfig() {
	v := viper.New()
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
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

func init() {
	parseConfig()
}
