package svc

import (
	"errors"
	"sort"
	"sync"
)

type Group struct {
	Name     string                    `json:"name"`
	Status   int                       `json:"status"`
	Cap      uint64                    `json:"cap"`
	Storages map[string]*StorageServer `json:"storages"`
	mu       sync.RWMutex
}

type StorageServer struct {
	Group      string `json:"group"`
	Addr       string `json:"addr"`
	Status     int    `json:"status"`
	Free       uint64 `json:"free"`
	UpdateTime int64  `json:"update_time"`
}

//GetStorages 获取注册的storage节点 map无法保证顺序
func (g *Group) GetStorages() []*StorageServer {
	g.mu.RLock()
	defer g.mu.RUnlock()
	keys := make([]string, 0)
	servers := make([]*StorageServer, 0)
	for _, server := range g.Storages {
		keys = append(keys, server.Addr)
	}
	//按addr排序
	sort.Strings(keys)
	for _, key := range keys {
		servers = append(servers, g.Storages[key])
	}
	return servers
}

func (g *Group) GetStorage(addr string) *StorageServer {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if val, ok := g.Storages[addr]; ok {
		return val
	}
	return nil
}

//RegisterStorage 注册storage节点
func (g *Group) RegisterStorage(s *StorageServer) error {
	if s.Addr == "" {
		return errors.New("storage addr can not be nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.Storages[s.Addr]; !ok {
		g.Storages[s.Addr] = s
	}
	return nil
}

//RemoveStorage 移除storage节点
func (g *Group) RemoveStorage(s *StorageServer) error {
	if s.Addr == "" {
		return errors.New("storage addr can not be nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.Storages, s.Addr)
	return nil
}

//UpdateStorage 更新storage节点
func (g *Group) UpdateStorage(s *StorageServer) error {
	if s.Addr == "" {
		return errors.New("storage addr can not be nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.Storages[s.Addr]; !ok {
		return errors.New("nil storage")
	}
	g.Storages[s.Addr] = s
	return nil
}

//IsExistStorage storage节点是否注册
func (g *Group) IsExistStorage(s StorageServer) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.Storages[s.Addr]
	return ok
}

//SaveOrUpdateStorage 注册或更新storage节点
func (g *Group) SaveOrUpdateStorage(s *StorageServer) error {
	if s.Addr == "" {
		return errors.New("storage addr can not be nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Storages[s.Addr] = s
	return nil
}

//GetGroupCap 获取group容量 cap==minimum storage free
func (g *Group) GetGroupCap() uint64 {
	g.mu.RLock()
	if len(g.Storages) == 0 {
		return 0
	}
	var caps []uint64
	for _, v := range g.Storages {
		caps = append(caps, v.Free)
	}
	g.mu.RUnlock()
	sort.Slice(caps, func(i, j int) bool {
		return caps[i] < caps[j]
	})
	return caps[0]
}
