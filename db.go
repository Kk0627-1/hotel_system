package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DBConfig struct{
	Host string
	Port int
	User string
	Password string
	DBName string
}

var defaultDBConfig = DBConfig{
	Host:     "localhost",
	Port:     5432,
	User:     "hotel_Manager",
	Password: "k0627123",
	DBName:   "hotel_manager",
}

var db *sql.DB 

func initDB(dbConfig DBConfig) error {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	log.Printf("连接字符串: %s", connString)
	var err error
	db, err = sql.Open("postgres", connString)
	if err != nil {
		log.Printf("sql.Open 失败: %v", err)
		return err
	}
	log.Println("sql.Open 成功")

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("连接池参数已设置 (maxOpen=25, maxIdle=5, maxLifetime=5m)")

	// 测试连接
	log.Println("正在 Ping 数据库…")
	err = db.Ping()
	if err != nil {
		log.Printf("Ping 失败，数据库不可达: %v", err)
		return err
	}
	log.Printf("数据库连接成功: dbname=%s host=%s port=%d", dbConfig.DBName, dbConfig.Host, dbConfig.Port)

	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}