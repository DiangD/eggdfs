package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"eggdfs/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	storageDBFileName = "storage.db"
)

type Storage struct {
	db *model.EggDB
}

type StorageStatus struct {
	Group  string `json:"group"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	Free   uint64 `json:"free"`
	Active bool   `json:"active"`
}

func NewStorage() *Storage {
	return &Storage{
		db: model.NewEggDB(storageDBFileName),
	}
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
	uuid := c.GetHeader(common.HeaderUUIDFileName)
	fileName := util.GenFileName(uuid, file.Filename)
	fullPath := baseDir + "/" + fileName
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
		FileId: uuid,
		Name:   file.Filename,
		ReName: fileName,
		Url:    "",
		Path:   fullPath,
		Md5:    md5hash,
		Size:   file.Size,
		Group:  config().Storage.Group,
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

//Download 下载 example todo
func (s *Storage) Download(c *gin.Context) {
	c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "goland.exe")) //fmt.Sprintf("attachment; filename=%s", filename)对下载的文件重命名
	c.Writer.Header().Add("Content-Type", "application/octet-stream")
	c.File("meta/2021/2/20/2000212.exe")
}

//Status 向tracker回报状态
func (s *Storage) Status() {
	c := config()
	status := &StorageStatus{
		Group:  c.Storage.Group,
		Host:   c.Host,
		Port:   c.Port,
		Active: true,
	}
	if stat, err := disk.Usage(c.Storage.StorageDir); err != nil {
		status.Free = 0
		status.Active = false
	} else {
		status.Free = stat.Free
	}

	for _, url := range c.Storage.Trackers {
		go func(url string) {
			_, _ = util.HttpPost(url+"/status", status, nil, time.Second)
		}(url)
	}
}

//startTimerTask 启动定时任务
func (s *Storage) startTimerTask() error {
	cr := cron.New(cron.WithSeconds())
	//5s per
	_, err := cr.AddFunc("*/5 * * * * *", func() {
		s.Status()
	})
	if err != nil {
		return err
	}
	cr.Start()
	return nil
}

//Start 启动Storage服务
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
	r.GET("/download", s.Download)
	r.Group("/v1")
	{
		r.POST("/upload", s.QuickUpload)
	}

	//开启定时任务
	if err := s.startTimerTask(); err != nil {
		logger.Panic("Storage定时任务启动失败", zap.String("addr", config().Host))
	}

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("Storage服务启动失败", zap.String("addr", config().Host))
	}
}
