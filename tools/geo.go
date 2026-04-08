package tools

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GetCity 根据IP地址查询对应的地理位置信息
// 参数：
//   path: GeoIP数据库文件路径
//   ipAddress: 要查询的IP地址字符串
// 返回值：
//   第一个返回值: 国家英文名称
//   第二个返回值: 城市英文名称
//   当出现错误时(如文件打开失败、IP格式错误)，返回空字符串
func GetCity(path, ipAddress string) (string, string) {
	// 打开GeoIP数据库文件
	db, err := geoip2.Open(path)
	if err != nil {
		// 数据库打开失败，返回空字符串
		return "", ""
	}
	// 确保函数结束时关闭数据库连接
	defer db.Close()
	
	// 解析IP地址并查询城市信息
	record, err := db.City(net.ParseIP(ipAddress))
	// 打印城市英文名称（调试信息）
	fmt.Println(record.City.Names["en"])
	// 返回国家英文名称和城市英文名称
	return record.Country.Names["en"], record.City.Names["en"]
}
