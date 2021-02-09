package svc

import (
	"eggdfs/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path/filepath"
)

type Storage struct {
}

func NewStorage() *Storage {
	return &Storage{}
}

type FileInfo struct {
	Name   string
	Rename string
	MD5    string
	Size   int64
}

type RespResult struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func hello(c *gin.Context) {
	c.JSON(http.StatusOK, RespResult{
		Status:  200,
		Message: "hello eggdfs!",
		Data:    nil,
	})
}

func simpleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, RespResult{
			Status:  400,
			Message: "参数解析失败",
			Data:    err.Error(),
		})
	}
	err = c.SaveUploadedFile(file, config().Storage.StorageDir)
	if err != nil {
		c.JSON(http.StatusOK, RespResult{
			Status:  400,
			Message: "文件保存失败",
			Data:    err.Error(),
		})
	}
	c.JSON(http.StatusOK, RespResult{
		Status:  200,
		Message: "文件上传成功",
	})
}

func (s *Storage) Start() {
	sd := config().Storage.StorageDir
	if _, err := os.Stat(sd); err != nil {
		err := os.MkdirAll(sd, os.ModePerm)
		path, _ := filepath.Abs(config().Storage.StorageDir)
		if err != nil {
			logger.Error("文件保存路径创建失败", zap.String("storage_dir", path))
			//todo tell to tracker
		}
		logger.Info("文件保存路径创建成功", zap.String("storage_dir", path))
	}

	r := gin.Default()
	//gin.SetMode(gin.ReleaseMode)

	r.GET("/hello", hello)
	r.Group("/v1")
	{
		r.GET("/upload", simpleUpload)
	}

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("Storage服务启动失败", zap.String("addr", config().Host))
	}
}
