package main

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

//db 全局数据库对象
var db *sql.DB

//Init 初始化数据库, 当数据库文件不存在时将创建一个默认的数据库文件
func init() {
	// 检查本地数据库是否存在
	if !Exists(SqlPath) {
		log.Println("数据库文件不存在, 将在默认位置初始化数据库文件")
		InitializeDB()
	}
	var err error
	// charset=utf 用于指示打开/新建文件时使用的字符编码类型
	db, err = sql.Open("sqlite3", SqlPath+"?charset=utf")
	if err != nil {
		log.Fatalf("打开数据库错误: %s", err)
	}

	// 尝试与数据库建立连接
	err = db.Ping()
	if err != nil {
		log.Fatalf("连接数据库错误: %s", err)
	}
}

//InitializeDB 创建默认数据库
func InitializeDB() {
	// 创建数据库文件
	db, err := sql.Open("sqlite3", SqlPath+"?charset=utf")
	if err != nil {
		log.Fatalf("打开数据库错误: %s", err)
	}
	defer db.Close()

	// 连接数据库文件
	err = db.Ping()
	if err != nil {
		log.Fatalf("连接数据库错误: %s", err)
	}

	// 创建各表
	// 表: Server
	sql_table := `
    CREATE TABLE IF NOT EXISTS Server(
		server_name TEXT NULL,
		uuid TEXT NULL,
		public_key TEXT NULL
	);
	`
	db.Exec(sql_table)

	// 表: Submission
	sql_table = `
    CREATE TABLE IF NOT EXISTS Submission(
		uuid TEXT NULL,
		player_uuid TEXT NULL,
		comment TEXT NULL,
		point INTEGER NULL
	);
	`
	db.Exec(sql_table)
}

// registerServer 存储注册服务器时返回的uuid和服务器名称
func registerServer(server_name string, uuid string) error {
	pubkey, err := ioutil.ReadFile("rsa-pub.pem")
	if err != nil {
		return errors.New("无法读取本地公钥: " + err.Error())
	}
	_, err = db.Exec("INSERT INTO Server (server_name, uuid, public_key) values(?,?,?)", server_name, uuid, pubkey)
	if err != nil {
		return err
	}
	return nil
}

// newSubmission 向数据库中存入提交记录
func newSubmission(uuid, player_uuid, comment string, point int) error {
	_, err := db.Exec("INSERT INTO Submission (uuid, player_uuid, comment, point) values(?,?,?,?)", uuid, player_uuid, comment, point)
	if err != nil {
		return err
	}
	return nil
}
