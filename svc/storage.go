package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"eggdfs/util"
	"encoding/json"
	"errors"
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
	"strconv"
	"time"
)

const (
	storageDBFileName = "storage"
)

type Storage struct {
	db       *model.EggDB
	trackers []string
}

type StorageStatus struct {
	Group string `json:"group"`
	Host  string `json:"host"`
	Port  string `json:"port"`
	Free  uint64 `json:"free"`
}

func NewStorage() *Storage {
	return &Storage{
		db:       model.NewEggDB(storageDBFileName),
		trackers: config().Storage.Trackers,
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
	fileHash := c.GetHeader(common.HeaderFileHash)
	logger.Info(fileHash)
	//秒传 检查数据库是否存在相同的md5
	if fileHash != "" {
		fi := model.FileInfo{}
		if exist, _ := s.db.IsExistKey(fileHash); exist {
			data, _ := s.db.Get(fileHash)
			_ = json.Unmarshal(data, &fi)
			c.JSON(http.StatusOK, model.RespResult{
				Status:  common.Success,
				Message: "文件已存在，妙传成功",
				Data:    fi,
			})
			return
		}
	}

	customDir := c.GetHeader(common.HeaderUploadFileDir)
	path := util.GenFilePath(customDir)
	baseDir := config().Storage.StorageDir + "/" + path
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
	uuid := c.GetHeader(common.HeaderFileUUID)
	fileName := util.GenFileName(uuid, file.Filename)
	fullPath := baseDir + "/" + fileName
	md5hash, err := s.SaveQuickUploadedFile(file, fullPath, fileHash)
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.FileSaveFail,
			Message: "文件保存失败",
			Data:    err.Error(),
		})
		return
	}
	fi := model.FileInfo{
		FileId: uuid,
		Name:   file.Filename,
		ReName: fileName,
		Url:    "",
		Path:   fmt.Sprintf("%s/%s", path, fileName),
		Md5:    md5hash,
		Size:   file.Size,
		Group:  config().Storage.Group,
	}
	bytes, _ := json.Marshal(fi)
	_ = s.db.Put(fi.Md5, bytes)
	c.Writer.Header().Set(common.HeaderFileUploadRes, strconv.Itoa(common.Success))
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "文件保存成功",
		Data:    fi,
	})
}

func (s *Storage) SaveQuickUploadedFile(file *multipart.FileHeader, dst string, hash string) (md5hash string, err error) {
	src, err := file.Open()
	if err != nil {
		return
	}
	defer src.Close()
	md5hash = util.GenMD5(src)
	//检查文件完整性
	if hash != md5hash && hash != "" {
		err = errors.New("file is already damaged")
		return
	}
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	if err != nil {
		return
	}
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
		Group: c.Storage.Group,
		Host:  c.Host,
		Port:  c.Port,
	}
	if stat, err := disk.Usage(c.Storage.StorageDir); err != nil {
		status.Free = 0
	} else {
		status.Free = stat.Free
	}

	for _, url := range s.trackers {
		go func(url string) {
			logger.Info("report to tracker", zap.String("tracker", url), zap.String("host", c.Host))
			_, _ = util.HttpPost(url+"/status", status, nil, time.Second)
		}(url)
	}
}

//Sync 文件同步
func (s *Storage) Sync(c *gin.Context) {
	var sync model.SyncFileInfo
	var syncFunc SyncFunc
	_ = c.ShouldBindJSON(&sync)

	if sync.Action == common.SyncAdd {
		syncFunc = s.SyncFileAdd
	}

	if sync.Action == common.SyncDelete {

	}

	if syncFunc != nil {
		syncFunc(sync, c)
	}
}

//SyncFunc 同步函数
type SyncFunc func(model.SyncFileInfo, *gin.Context)

func (s *Storage) SyncFileAdd(sync model.SyncFileInfo, c *gin.Context) {
	base := config().Storage.StorageDir + sync.FilePath
	if _, err := os.Stat(base); err != nil {
		err := os.MkdirAll(base, os.ModePerm)
		if err != nil {
			//todo report to tracker
			c.JSON(http.StatusOK, model.RespResult{
				Status: common.Fail,
			})
		}
	}

	//download file
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", sync.Src, sync.Group, sync.FileName))
	if err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status: common.Fail,
		})
	}
	f, err := os.Create(base + sync.FileName)
	if err != nil {
		//todo report to tracker
		c.JSON(http.StatusOK, model.RespResult{
			Status: common.Fail,
		})
	}
	l, err := io.Copy(f, resp.Body)
	if err != nil || l <= 0 {
		//todo report to tracker
		c.JSON(http.StatusOK, model.RespResult{
			Status: common.Fail,
		})
	}
	defer resp.Body.Close()
	//todo add db log
	c.JSON(http.StatusOK, model.RespResult{
		Status: common.Success,
	})
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

	r.StaticFS(conf.Config().Storage.Group, http.Dir(config().Storage.StorageDir))

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

	err := r.Run(":" + config().Port)
	if err != nil {
		logger.Panic("Storage服务启动失败", zap.String("addr", config().Host))
	}
}
