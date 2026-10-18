package main

import (
	"log"
	"net/http"
)

type Config struct{}

const port = "8080"

func main() {
	app := Config{}
	_ = app
	// 初始化数据库
	if err := initDB(defaultDBConfig); err != nil {
		log.Fatal(err)
	}
	log.Println("应用初始化完成")
	defer CloseDB()

	// 初始化表
	if err := InitUserTable(); err != nil {
		log.Fatal(err)
	}
	if err := InitWalletTable(); err != nil {
		log.Fatal(err)
	}
	if err := InitHotelTable(); err != nil {
		log.Fatal(err)
	}
	if err := InitRoomTable(); err != nil {
		log.Fatal(err)
	}
	if err := InitRecordTable(); err != nil {
		log.Fatal(err)
	}
	log.Println("表初始化完成")
	log.Printf("正在启动服务器，端口: %s", port)
	log.Fatal(http.ListenAndServe(":"+port, app.routes()))
}