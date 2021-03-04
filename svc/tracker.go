package svc

import (
	"eggdfs/common"
	"eggdfs/common/model"
	"eggdfs/logger"
	"eggdfs/util"
	"encoding/json"
	"errors"
	"fmt"
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
	syncDB *model.EggDB //sync err log
	hash   Hash
	mu     sync.RWMutex //map mutex
	lock   sync.Mutex   //process mutex
}

//NewTracker 构造函数可使用自定义的hash
func NewTracker(fn Hash) *Tracker {
	t := &Tracker{
		groups: make(map[string]*Group),
		syncDB: model.NewEggDB("sync-err"),
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

	//report storage status
	r.POST("/status", t.StorageStatusReport)
	r.Group("/v1")
	{
		//upload file
		r.POST("/upload", t.QuickUpload)
	}
	//delete file
	r.POST("/delete", t.Delete)
	//get group status
	r.GET("/g/status", t.GroupStatus)
	//sync err-log from storage
	r.POST("/err/log", t.SyncErrorMsg)

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
	//储存空间阈值
	if sm.Free <= common.MinStorageSpace {
		sm.Status = common.StorageNotEnoughSpace
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
	group := t.GetGroup(sm.Group)
	if group == nil {
		return
	}
	if err := group.SaveOrUpdateStorage(sm); err != nil {
		return
	}
	t.lock.Unlock()

	go func() {
		t.SetTrackerStatus()
	}()
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

//GetGroups 获取所有group
func (t *Tracker) GetGroups() []*Group {
	var gs []*Group
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, group := range t.groups {
		gs = append(gs, group)
	}
	return gs
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

//QuickUpload api 小文件快传
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

//SyncFile 同步函数
func (t *Tracker) SyncFile(sm *StorageServer, sync model.SyncFileInfo) {
	data, err := json.Marshal(sync)
	if err != nil {
		return
	}
	if sm.Status != common.StorageActive {
		_ = t.syncDB.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
	url := sm.HttpSchema + "://" + sm.Addr + "/sync"
	//在大文件下这个方法完全不可行
	resp, err := util.HttpPost(url, sync, nil, time.Second*30)
	if err != nil || len(resp) == 0 {
		_ = t.syncDB.Put(sync.FileName+"@"+sync.Action, data)
		return
	}

	//res code写入日志
	var res model.RespResult
	err = json.Unmarshal(resp, &res)
	if err != nil {
		_ = t.syncDB.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
	if res.Status != common.Success {
		_ = t.syncDB.Put(sync.FileName+"@"+sync.Action, data)
		return
	}
}

//Delete api 文件删除
func (t *Tracker) Delete(c *gin.Context) {
	var deleteFile struct {
		FileID string `json:"file_id" form:"file_id"`
		Group  string `json:"group" form:"group" binding:"required"` // group required
		MD5    string `json:"md5" form:"md5"`
		File   string `json:"file" form:"file" binding:"required"` //path required
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
				_ = t.syncDB.Put(strings.Join([]string{info.FileName, info.Dst, common.SyncDelete}, "@"), data)
				return
			}

			res, err := util.HttpPost(url, info, nil, time.Second*10)
			if err != nil {
				_ = t.syncDB.Put(info.FileName+"@"+common.SyncDelete, data)
				return
			}
			var resp model.RespResult
			_ = json.Unmarshal(res, &resp)

			//error log
			if resp.Status != common.Success {
				_ = t.syncDB.Put(strings.Join([]string{info.FileName, info.Dst, common.SyncDelete}, "@"), data)
			}
		}()
	}
	c.JSON(http.StatusOK, model.RespResult{
		Status:  common.Success,
		Message: "删除成功",
	})
}

//SetTrackerStatus 计算整个tracker以及所有group的状态
func (t *Tracker) SetTrackerStatus() {
	t.lock.Lock()
	defer t.lock.Unlock()
	gs := t.GetGroups()
	for _, group := range gs {
		vs := make([]*StorageServer, 0)
		for _, s := range group.GetStorages() {
			if s.Status == common.StorageActive {
				//10次没有接收到storage report，认为已离线
				if time.Now().Unix()-s.UpdateTime > 5*10 {
					s.Status = common.StorageOffline
				} else {
					vs = append(vs, s)
				}
			}
		}
		//没有可用的storage
		if len(vs) == 0 {
			group.Status = common.GroupUnavailable
			group.Cap = 0
		} else {
			group.Status = common.GroupActive
			group.Cap = group.GetGroupCap()
		}
		logger.Info("Tracker & Group status", zap.Any("tracker", t.groups))
	}
}

//GroupStatus api 获取集群状态
func (t *Tracker) GroupStatus(c *gin.Context) {
	g := c.Query("group")
	if g != "" {
		group := t.GetGroup(g)
		if group != nil {
			c.JSON(http.StatusOK, model.RespResult{
				Status: common.Success,
				Data:   group,
			})
			return
		}
		c.JSON(http.StatusOK, model.RespResult{
			Status:  common.Fail,
			Message: "no such group",
		})
		return
	}
	groups := t.GetGroups()
	c.JSON(http.StatusOK, model.RespResult{
		Status: common.Success,
		Data:   groups,
	})
}

//SyncErrorMsg 同步storage的err log
func (t *Tracker) SyncErrorMsg(c *gin.Context) {
	var errMsg struct {
		ErrCode int
		Group   string
		Host    string
		Port    string
		ErrMsg  string
	}

	_ = c.ShouldBindJSON(&errMsg)
	if errMsg.ErrMsg != "" {
		addr := net.JoinHostPort(errMsg.Host, errMsg.Port)
		//同步到tracker日志中，方便排查
		logger.Error(fmt.Sprintf("sync err log => errcode:%d,msg:%s", errMsg.ErrCode, errMsg.ErrMsg),
			zap.String("addr", addr),
			zap.String("group", errMsg.Group))
	}
}
