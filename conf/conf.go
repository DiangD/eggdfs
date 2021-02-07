package conf

import (
	"eggFs/logger"
	"encoding/json"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"sync/atomic"
	"unsafe"
)

const (
	ConfigFilePath = "config.json"
)

var configPtr unsafe.Pointer

type GlobalConfig struct {
	Port    string `json:"port"`
	Host    string `json:"host"`
	Group   string `json:"group"`
	DataDir string `json:"data_dir"`
}

func ParseConfig(path string) {
	file, err := os.Open(path)
	if err != nil {
		logger.Panic("配置文件不存在", zap.String("path", path))
	}
	defer func() { _ = file.Close() }()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Panic("配置文件读取失败", zap.String("path", path))
	}
	var c GlobalConfig
	if err = json.Unmarshal(data, &c); err != nil {
		logger.Panic("配置文件解析失败", zap.String("path", path))
	}
	logger.Info("配置文件解析成功", zap.Any("conf", c))
	//并发安全，指针赋值
	atomic.StorePointer(&configPtr, unsafe.Pointer(&c))
}

func Config() *GlobalConfig {
	return (*GlobalConfig)(atomic.LoadPointer(&configPtr))
}
