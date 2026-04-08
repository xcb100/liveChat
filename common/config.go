package common

import (
	"encoding/json"
	"goflylivechat/tools"
	"io/ioutil"
)

// Mysql 配置结构体
type Mysql struct {
	Server   string
	Port     string
	Database string
	Username string
	Password string
}

// Redis 配置结构体
type Redis struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

// GetMysqlConf 获取MySQL配置
func GetMysqlConf() *Mysql {
	var mysql = &Mysql{}
	isExist, _ := tools.IsFileExist(MysqlConf)
	if !isExist {
		return mysql
	}
	info, err := ioutil.ReadFile(MysqlConf)
	if err != nil {
		return mysql
	}
	err = json.Unmarshal(info, mysql)
	return mysql
}

// GetRedisConf 获取Redis配置
func GetRedisConf() *Redis {
	var redis = &Redis{
		// 默认值
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
	isExist, _ := tools.IsFileExist(RedisConf)
	if !isExist {
		return redis
	}
	info, err := ioutil.ReadFile(RedisConf)
	if err != nil {
		return redis
	}
	err = json.Unmarshal(info, redis)
	return redis
}
