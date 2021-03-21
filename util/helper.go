package util

import (
	"bytes"
	"context"
	"crypto/md5"
	"eggdfs/common"
	"eggdfs/logger"
	"eggdfs/svc/conf"
	"encoding/hex"
	"encoding/json"
	"github.com/bwmarrin/snowflake"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

//DNode 全局雪花算法Node
var DNode *snowflake.Node

func init() {
	GenTrackerNodeId()
}

//GenTrackerNodeId 生成唯一Tracker id
func GenTrackerNodeId() {
	var err error
	if conf.Config().Tracker.NodeId > 0 {
		DNode, err = snowflake.NewNode(conf.Config().Tracker.NodeId)
	}
	rand.Seed(time.Now().UnixNano())
	DNode, err = snowflake.NewNode(int64(rand.Intn(1024)))
	if err != nil {
		logger.Panic("Error Tracker NodeId")
	}
}

//GenFilePath 生成文件路径
func GenFilePath(paths ...string) (root string) {
	var p string
	if len(paths) > 0 {
		p = paths[0]
	}
	cur := time.Now()
	year, month, day := cur.Year(), cur.Month(), cur.Day()
	root = strings.Join([]string{strconv.Itoa(year),
		strconv.Itoa(int(month)),
		strconv.Itoa(day),
	}, "/")
	if p != "" {
		root = root + "/" + p
	}
	return root
}

//GenUUIDFileName 生成文件名
func GenFileUUID() string {
	return DNode.Generate().String()
}

func GenFileName(id, fn string) (fileName string) {
	return id + path.Ext(fn)
}

//GenMD5 生成文件md5 不适合大文件
func GenMD5(src io.Reader) (string, error) {
	md5h := md5.New()
	if _, err := io.Copy(md5h, src); err != nil {
		return "", nil
	}
	return hex.EncodeToString(md5h.Sum([]byte(""))), nil
}

//HttpPost 发送http post请求
func HttpPost(url string, data interface{}, header map[string]string, timeout time.Duration) (res []byte, err error) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return
	}
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.DefaultClient
	client.Timeout = timeout
	resp, err := client.Do(req.WithContext(context.TODO()))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	res, err = ioutil.ReadAll(resp.Body)
	return res, err
}

//ParseHeaderFilePath 解析路径与文件名 eq path.Dir path.Base
func ParseHeaderFilePath(path string) (filePath, filename string) {
	index := strings.LastIndex(path, "/")
	return path[:index], path[index+1:]
}

func GetFileContentType(ext string) string {
	ext = strings.ToLower(ext)
	if val, ok := common.FileContentType[ext]; ok {
		if val != "" {
			return val
		}
	}
	return common.DefaultFileDownloadContentType
}
