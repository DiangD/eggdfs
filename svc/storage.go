package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"eggdfs/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
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
		Status:  common.Success,
		Message: "hello eggdfs storage!",
		Data:    nil,
	})
}

//QuickUpload 适合小文件
func (s *Storage) QuickUpload(c *gin.Context) {
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
	//文件名由雪花算法的服务器生成
	uuidFilename := c.GetHeader(common.HeaderUUIDFileName)
	fullPath := baseDir + "/" + util.GenFileName(uuidFilename, file.Filename)
	md5hash, err := s.SaveQuickUploadedFile(file, fullPath)
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
		ReName: uuidFilename,
		Size:   file.Size,
		Group:  config().Storage.Group,
		Md5:    md5hash,
	}
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "文件保存成功",
		Data:    fi,
	})
}

func (s *Storage) SaveQuickUploadedFile(file *multipart.FileHeader, dst string) (md5hash string, err error) {
	src, err := file.Open()
	if err != nil {
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	if err != nil {
		return
	}
	md5hash = util.GenFileMD5(src)
	return md5hash, nil
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
		r.POST("/upload", s.QuickUpload)
	}

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("Storage服务启动失败", zap.String("addr", config().Host))
	}
}
