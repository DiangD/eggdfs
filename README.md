# egg-dfs

一个简易的分布式文件系统，使用Go语言开发，具有部署简单，配置简明的特点。缺点也很多，还在不断优化。

设计参考了`go-dfs` 、`fastdfs`等开源项目，后端主要包括tracker、storage 两个角色。

```
客户端 ---> tracker------>storage1
                 \------>storage2
```

## 功能说明

* 文件上传（秒传）
* 文件下载
* 文件删除
* 文件多副本同步保存和删除
* 分组容量负载均衡

### 配置文件
示例：
```json
{
  "env": "debug", 
  "deploy_type": "tracker",
  "http_schema": "http",
  "port": "8081",
  "host": "127.0.0.1",
  "log_dir": "./log/zap.log",
  "tracker": {
    "node_id": "1000",
    "enable_tmp_file": true
  },
  "storage": {
    "group": "g1",
    "file_size_limit": -1,
    "storage_dir": "./meta",
    "trackers": [
      "http://127.0.0.1:9000",
      "http://127.0.0.1:8081"
    ]
  }
}
```
说明：
```json
{
  "env": "部署的环境 debug:日志输出到控制台 prod:日志输出到如下log_dir", 
  "deploy_type": "部署服务类型 tracker||storage",
  "http_schema": "http",
  "port": "端口",
  "host": "IP",
  "log_dir": "日志储存位置 ./log/zap.log",
  "tracker": {
    "node_id": "节点ID 可填也可自动生成",
    "enable_tmp_file": true
  },
  "storage": {
    "group": "group名称 g1",
    "file_size_limit": -1 ,
    "storage_dir": "文件保存路径 ./meta",
    "trackers": [
      "tracker的网址",
      "http://127.0.0.1:9000",
      "http://127.0.0.1:8081"
    ]
  }
}
```

## 使用的技术
* gin，高性能的web框架
* leveldb，基于golang的kv数据库
* viper，配置文件框架
* zap，高性能日志框架
* cron，定时任务

## todo待续
- [ ] 大文件分片上传  
- [ ] 优化文件同步逻辑
- [ ] 临时文件上传
- [ ] 断点续传下载