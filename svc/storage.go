package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"eggdfs/util"
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

func hello(c *gin.Context) {
	c.JSON(http.StatusOK, model.RespResult{
		Status:  200,
		Message: "hello eggdfs!",
		Data:    nil,
	})
}

func simpleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FormFileNotFound,
			Message: "参数解析失败",
			Data:    err.Error(),
		})
	}
	err = c.SaveUploadedFile(file, config().Storage.StorageDir)
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FileSaveFail,
			Message: "文件保存失败",
			Data:    err.Error(),
		})
	}
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "文件上传成功",
	})
}

func (s *Storage) Upload(c *gin.Context) {
	//用户自定义的存储文件夹
	customDir := c.GetHeader(common.HeaderUploadFileDir)
	baseDir := config().Storage.StorageDir + "/" + util.GenFilePath(customDir)
	if _, err := os.Stat(baseDir); err != nil {
		err := os.MkdirAll(baseDir, os.ModePerm)
		path, _ := filepath.Abs(config().Storage.StorageDir)
		if err != nil {
			logger.Error("文件保存路径创建失败", zap.String("file_baseDir", path))
			//todo tell to tracker
			c.JSON(http.StatusOK, model.RespResult{
				Status:  common.DirCreateFail,
				Message: "文件保存路径创建失败",
				Data:    nil,
			})
			return
		}
		logger.Info("文件保存路径创建成功", zap.String("file_baseDir", path))
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FormFileNotFound,
			Message: "未能索引上传文件",
			Data:    nil,
		})
		return
	}

	//文件大小限制
	if config().Storage.FileSizeLimit > 0 && file.Size > config().Storage.FileSizeLimit {
		logger.Warn("文件大小超过限制", zap.String("file", file.Filename), zap.Int64("size", file.Size))
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FileSizeExceeded,
			Message: "文件大小超过限制",
			Data:    nil,
		})
		return
	}

	//保存文件
	fullPath := baseDir + "/" + util.GenFileName(c.GetHeader(common.HeaderUUIDFileName), file.Filename)
	err = c.SaveUploadedFile(file, fullPath)
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FileSaveFail,
			Message: "文件保存失败",
			Data:    err,
		})
		return
	}
	fi := model.FileInfo{
		Name:   file.Filename,
		ReName: c.GetHeader(common.HeaderUUIDFileName),
		Size:   file.Size,
		Group:  config().Storage.Group,
	}
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "文件保存成功",
		Data:    fi,
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

	r.StaticFS(conf.Config().Storage.Group+"/static", http.Dir(config().Storage.StorageDir))

	r.GET("/hello", hello)
	r.Group("/v1")
	{
		r.POST("/upload", s.Upload)
	}

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("Storage服务启动失败", zap.String("addr", config().Host))
	}
}
