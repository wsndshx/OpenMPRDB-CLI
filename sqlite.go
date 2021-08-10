package main

import (
	"database/sql"
	"errors"
	"fmt"
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
		uuid TEXT NULL UNIQUE,
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
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}

// newSubmission 向数据库中存入提交记录
func newSubmission(uuid, player_uuid, comment string, point int) error {
	_, err := db.Exec("INSERT INTO Submission (uuid, player_uuid, comment, point) values(?,?,?,?)", uuid, player_uuid, comment, point)
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}

// deleteSubmission 删除数据库中指定的提交记录
func deleteSubmission(uuid string) error {
	_, err := db.Exec("DELETE FROM Submission WHERE uuid = ?", uuid)
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}

// serverList 吐出数据库Server表中的部分内容
func serverList(c chan string) error {
	rows, err := db.Query("SELECT server_name, uuid FROM Server")
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var data struct {
			uuid string
			name string
		}
		err := rows.Scan(&data.name, &data.uuid)
		if err != nil {
			return errors.New("本地数据库错误: " + err.Error())
		}
		c <- fmt.Sprintf("%s\t|%s", data.uuid, data.name)
	}
	close(c)
	return nil
}

// subList 吐出数据库Submission表中的所有内容
func subList(c chan string) error {
	rows, err := db.Query("SELECT * FROM Submission")
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var data struct {
			uuid        string
			player_uuid string
			comment     string
			point       int
		}
		err := rows.Scan(&data.uuid, &data.player_uuid, &data.comment, &data.point)
		if err != nil {
			return errors.New("本地数据库错误: " + err.Error())
		}

		c <- fmt.Sprintf("%s\t|%s\t|   %d\t|%s", data.uuid, data.player_uuid, data.point, data.comment)
	}
	close(c)
	return nil
}

// insertServer 插入新的服务器信息
func insertServer(uuid, name, pubkey_path string) error {
	// 读取本地公钥
	pubkey, err := ioutil.ReadFile(pubkey_path)
	if err != nil {
		return errors.New("读取指定公钥错误: " + err.Error())
	}

	_, err = db.Exec("INSERT INTO Server (server_name, uuid, public_key) values(?,?,?)", name, uuid, string(pubkey))
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}
