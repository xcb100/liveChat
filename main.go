package main

import (
	"goflylivechat/cmd"
)

// main 输入命令行参数，输出为启动对应子命令，目的在于作为程序统一入口。
func main() {
	cmd.Execute()
}
