package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/util"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"hash/crc32"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Hash func([]byte) uint32

type Tracker struct {
	groups map[string]*Group
	db     *model.EggDB
	hash   Hash
	mu     sync.RWMutex //map mutex
	lock   sync.Mutex   //process mutex
}

//NewTracker 构造函数可使用自定义的hash
func NewTracker(fn Hash) *Tracker {
	t := &Tracker{
		groups: make(map[string]*Group),
		db:     model.NewEggDB("tracker"),
		hash:   fn,
	}
	if t.hash == nil {
		t.hash = crc32.ChecksumIEEE
	}
	return t
}

//Start 服务启动入口
func (t *Tracker) Start() {
	r := gin.Default()

	r.POST("/status", t.StorageStatusReport)
	r.Group("/v1")
	{
		r.POST("/upload", t.QuickUpload)
	}
	r.POST("/delete", t.Delete)

	err := r.Run(":" + config().Port)
	if err != nil {
		addr := net.JoinHostPort(config().Host, config().Port)
		logger.Panic("Tracker服务启动失败", zap.String("addr", addr))
	}
}

//startTrackerTimerTask tracker定时任务
func (t *Tracker) startTrackerTimerTask() error {
	cr := cron.New(cron.WithSeconds())

	cr.Start()
	return nil
}

//StorageStatusReport 处理storage回报的信息
func (t *Tracker) StorageStatusReport(c *gin.Context) {
	var params struct {
		Group      string `json:"group" binding:"required"`
		HttpSchema string `json:"http_schema" binding:"required"`
		Host       string `json:"host" binding:"required"`
		Port       string `json:"port" binding:"required"`
		Free       uint64 `json:"free" binding:"required"`
	}
	if err := c.ShouldBindJSON(&params); err != nil {
		logger.Error("param binding fail", zap.String("url", "/status"))
	}
	//nil addr
	if params.Port == "" || params.Host == "" {
		logger.Error("nil addr", zap.String("url", "/status"))
		return
	}

	//nil group
	if params.Group == "" {
		logger.Error("nil group", zap.String("group", "/status"))
		return
	}

	sm := &StorageServer{
		Group:      params.Group,
		HttpSchema: params.HttpSchema,
		Addr:       net.JoinHostPort(params.Host, params.Port),
		Free:       params.Free,
		Status:     common.StorageActive,
		UpdateTime: time.Now().Unix(),
	}
	//储存空间阈值 //todo
	if sm.Free <= 100000 {
		return
	}

	t.lock.Lock()
	//防止重复注册，覆盖注册的情况
	//group未注册
	if !t.IsExistGroup(sm.Group) {
		group := &Group{
			Name:     sm.Group,
			Status:   common.GroupActive,
			Cap:      sm.Free,
			Storages: make(map[string]*StorageServer),
		}
		_ = group.SaveOrUpdateStorage(sm)
		_ = t.RegisterGroup(group)
	}

	//group已注册
	//todo 思路：应该先添加storage还是先计算cap... 逻辑：是否及时计算cap，以及group状态
	group := t.GetGroup(sm.Group)
	if group == nil {
		return
	}
	if err := group.SaveOrUpdateStorage(sm); err == nil {
		group.Cap = group.GetGroupCap()
	}
	t.lock.Unlock()
	logger.Info("tracker group details", zap.Any("groups", t.groups))
}

//SelectStorageIPHash ip hash选择storage
func (t *Tracker) SelectStorageIPHash(ip string, g *Group) (s *StorageServer, err error) {
	as := make([]*StorageServer, 0)
	for _, g := range g.GetStorages() {
		if g.Status == common.StorageActive {
			as = append(as, g)
		}
	}
	if len(as) == 0 {
		return nil, errors.New("no available storage for group")
	}
	//ip hash
	hashcode := int(t.hash([]byte(ip)))
	return as[hashcode%len(as)], err
}

//GetGroup group name 获取group
func (t *Tracker) GetGroup(name string) *Group {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.groups[name]
}

//IsExistGroup group是否注册
func (t *Tracker) IsExistGroup(groupName string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.groups[groupName]
	return ok
}

//RegisterGroup 注册group
func (t *Tracker) RegisterGroup(g *Group) error {
	if g.Name == "" {
		return errors.New("group name can not be nil")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.groups[g.Name]; !ok {
		t.groups[g.Name] = g
	}
	return nil
}

//SelectGroupForUpload 选择上传的group
func (t *Tracker) SelectGroupForUpload() (*Group, error) {
	gs := make([]*Group, 0)
	t.mu.RLock()
	for _, g := range t.groups {
		if g.Status == common.GroupActive {
			gs = append(gs, g)
		}
	}
	t.mu.RUnlock()
	if len(gs) == 0 {
		return nil, errors.New("no available group")
	}

	//按容量降序
	sort.Slice(gs, func(i, j int) bool {
		return gs[i].Cap > gs[i].Cap
	})
	//返回容量最大的group
	return gs[0], nil
}

//QuickUpload 小文件快传
func (t *Tracker) QuickUpload(c *gin.Context) {
	//获取group
	group, err := t.SelectGroupForUpload()
	if err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}
	//获取storage
	s, err := t.SelectStorageIPHash(c.ClientIP(), group)
	if err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}

	//set header
	uuid := util.GenFileUUID()
	c.Request.Header.Set(common.HeaderFileUUID, uuid)

	//反向代理
	p := NewTrackerProxy(config().HttpSchema, s.Addr, group.Name, t, c)
	if err = t.httpProxy(p, c); err != nil {
		logger.Error(err.Error())
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: err.Error(),
		})
		return
	}

	//文件同步
	if c.Writer.Header().Get(common.HeaderFileUploadRes) == strconv.Itoa(common.Success) {
		fullPath := c.Writer.Header().Get(common.HeaderFilePath)
		hash := c.Writer.Header().Get(common.HeaderFileHash)
		filePath, filename := util.ParseHeaderFilePath(fullPath)
		storages := t.groups[group.Name].GetStorages()
		for i := 0; i < len(storages); i++ {
			server := storages[i]
			if server.Addr == s.Addr {
				continue
			}
			go func(*StorageServer) {
				info := model.SyncFileInfo{
					Src:      s.HttpSchema + "://" + s.Addr,
					Dst:      server.HttpSchema + "://" + server.Addr,
					FileId:   uuid,
					FilePath: filePath,
					FileName: filename,
					FileHash: hash,
					Action:   common.SyncAdd,
					Group:    group.Name,
				}
				logger.Info("sync-file info", zap.Any("info", info))
				t.SyncFile(server, info)
			}(server)
		}
	}
}

//httpProxy tracker 反向代理 //todo 处理502请求2 redo proxy
func (t *Tracker) httpProxy(tp *TrackerProxy, c *gin.Context) error {
	return tp.HttpProxy(c.Writer, c.Request, tp.AbortErrorHandler)
}

//SyncFile 同步函数 todo
func (t *Tracker) SyncFile(sm *StorageServer, sync model.SyncFileInfo) {
	data, err := json.Marshal(sync)
	if err != nil {
		return
	}
	if sm.Status != common.StorageActive {
		_ = t.db.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
	url := sm.HttpSchema + "://" + sm.Addr + "/sync"
	//在大文件下这个方法完全不可行
	resp, err := util.HttpPost(url, sync, nil, time.Second*30)
	if err != nil || len(resp) == 0 {
		_ = t.db.Put(sync.FileName+"@"+sync.Action, data)
		return
	}

	//res code写入日志
	var res model.RespResult
	err = json.Unmarshal(resp, &res)
	if err != nil {
		_ = t.db.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
	if res.Status != common.Success {
		_ = t.db.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
}

func (t *Tracker) Delete(c *gin.Context) {
	var deleteFile struct {
		FileID string `json:"file_id" form:"file_id"`
		Group  string `json:"group" form:"group"`
		MD5    string `json:"md5" form:"md5"`
		File   string `json:"file" form:"file"`
	}
	if err := c.ShouldBind(&deleteFile); err != nil {
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.ParamBindFail,
			Message: "参数绑定失败",
		})
	}
	logger.Info("delete info", zap.Any("info", deleteFile))
	g := t.GetGroup(deleteFile.Group)
	for _, s := range g.GetStorages() {
		server := s
		go func() {
			filePath, filename := util.ParseHeaderFilePath(deleteFile.File)
			info := model.SyncFileInfo{
				Dst:      server.HttpSchema + "://" + server.Addr,
				FileId:   deleteFile.FileID,
				FilePath: filePath,
				FileName: filename,
				FileHash: deleteFile.MD5,
				Action:   common.SyncDelete,
				Group:    g.Name,
			}
			url := server.HttpSchema + "://" + server.Addr + "/sync"
			data, _ := json.Marshal(info)

			//跳过下线主机
			if server.Status == common.StorageOffline {
				_ = t.db.Put(strings.Join([]string{info.FileName, info.Dst, common.SyncDelete}, "@"), data)
				return
			}

			res, err := util.HttpPost(url, info, nil, time.Second*10)
			if err != nil {
				_ = t.db.Put(info.FileName+"@"+common.SyncDelete, data)
				return
			}
			var resp model.RespResult
			_ = json.Unmarshal(res, &resp)
			if resp.Status != common.Success {
				_ = t.db.Put(strings.Join([]string{info.FileName, info.Dst, common.SyncDelete}, "@"), data)
			}
		}()
	}
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "删除成功",
	})
}
