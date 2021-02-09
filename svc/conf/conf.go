package conf

import (
	"github.com/spf13/viper"
	"sync/atomic"
	"unsafe"
)

const (
	defaultConfigFileName = "eggdfs_config"
)

var configPtr unsafe.Pointer

//GlobalConfig 全局配置
type GlobalConfig struct {
	DeployType string `mapstructure:"deploy_type"`
	Port       string `mapstructure:"port"`
	Host       string `mapstructure:"host"`
	LogDir     string `mapstructure:"log_dir"`

	//tracker配置
	Tracker struct {
		NodeId        string `mapstructure:"node_id"`
		EnableTmpFile bool   `mapstructure:"enable_tmp_file"`
	} `json:"tracker"`

	//storage配置
	Storage struct {
		Group         string   `mapstructure:"group"`
		FileSizeLimit int64    `mapstructure:"file_size_limit"`
		StorageDir    string   `mapstructure:"storage_dir"`
		Trackers      []string `mapstructure:"trackers"`
	} `json:"storage"`
}

//parseConfig 解析配置文件
func parseConfig() {
	v := viper.New()
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.SetConfigName(defaultConfigFileName)
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

//Config 获取配置参数
func Config() *GlobalConfig {
	return (*GlobalConfig)(atomic.LoadPointer(&configPtr))
}

func InitGlobalConfig() {
	parseConfig()
}

func init() {
	parseConfig()
}
