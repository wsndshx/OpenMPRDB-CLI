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

// ReportList
type ReportList struct {
	player_uuid string
	point       float64
}

// SubList 提交列表
type SubList struct {
	uuid        string
	player_uuid string
	comment     string
	point       float64
	level       int
}

type ServerList struct {
	uuid   string
	name   string
	pubkey string
	level  int
	end    bool
}

//Init 初始化数据库, 当数据库文件不存在时将创建一个默认的数据库文件
func init() {
	// 检查本地数据库是否存在
	if !Exists(SqlPath) {
		log.Println("数据库文件不存在, 将在默认位置初始化数据库文件")
		// 初始化本地密钥
		err := initializationKey()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("密钥文件生成成功, 请妥善保管相关副本")
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
		public_key TEXT NULL,
		level INTEGER NULL
	);
	`
	db.Exec(sql_table)

	// 表: Submission
	sql_table = `
    CREATE TABLE IF NOT EXISTS Submission(
		uuid TEXT NULL,
		player_uuid TEXT NULL,
		comment TEXT NULL,
		point REAL NULL
	);
	`
	db.Exec(sql_table)

	// 表: Config
	sql_table = `
    CREATE TABLE IF NOT EXISTS Config(
		server_name TEXT NULL,
		private_key TEXT NULL,
		public_key TEXT NULL,
		uuid TEXT NULL
	);
	`
	db.Exec(sql_table)

	// 表: Reputation
	sql_table = `
    CREATE TABLE IF NOT EXISTS Reputation(
		player_uuid TEXT NULL UNIQUE,
		point REAL NULL
	);
	`
	db.Exec(sql_table)

	// 读取私钥
	privkey, err := ioutil.ReadFile("rsa-priv.pem")
	if err != nil {
		log.Fatalf("获取私钥错误: %s", err)
	}
	// 读取公钥
	pubkey, err := ioutil.ReadFile("rsa-pub.pem")
	if err != nil {
		log.Fatalf("获取公钥错误: %s", err)
	}
	// 存储
	_, err = db.Exec("INSERT INTO Config (private_key, public_key) values(?,?)", privkey, pubkey)
	if err != nil {
		log.Fatalf("数据库错误: %s", err)
	}

}

// registerServer 存储注册服务器时返回的uuid和服务器名称
func registerServer(server_name string, uuid string) error {
	_, err := db.Exec("UPDATE Config SET server_name = ?, uuid = ?", server_name, uuid)
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}

// newSubmission 向数据库中存入提交记录
func newSubmission(uuid, player_uuid, comment string, point float64) error {
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
func serverList(c chan ServerList) {
	rows, err := db.Query("SELECT * FROM Server")
	if err != nil {
		close(c)
		log.Panicf("本地数据库错误: %s\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data ServerList
		err := rows.Scan(&data.name, &data.uuid, &data.pubkey, &data.level)
		if err != nil {
			close(c)
			log.Panicf("本地数据库错误: %s\n", err)
			return
		}
		c <- data
	}
	close(c)
	return
}

// subList 吐出数据库Submission表中的所有内容
func subList(c chan SubList) {
	rows, err := db.Query("SELECT * FROM Submission")
	if err != nil {
		close(c)
		log.Panicf("本地数据库错误: %s\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data SubList
		err := rows.Scan(&data.uuid, &data.player_uuid, &data.comment, &data.point)
		if err != nil {
			close(c)
			log.Panicf("本地数据库错误: %s\n", err)
			return
		}
		data.level = 5
		c <- data
	}
	close(c)
	return
}

// insertServer 插入新的服务器信息
func insertServer(uuid, name, pubkey_path string, level int) error {
	// 读取本地公钥
	pubkey, err := ioutil.ReadFile(pubkey_path)
	if err != nil {
		return errors.New("读取指定公钥错误: " + err.Error())
	}

	_, err = db.Exec("INSERT INTO Server (server_name, uuid, public_key, level) values(?,?,?,?)", name, uuid, string(pubkey), level)
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	return nil
}

// addReputation 插入玩家的声望数据
func addReputation(data SubList) {
	// 若表中存在该uuid, 则将将传入的数据与原先的数据相加; 否则插入新行
	rows := db.QueryRow("SELECT player_uuid FROM Reputation WHERE player_uuid GLOB '?'", data.player_uuid)
	if rows.Scan().Error() == "" {
		_, err := db.Exec("UPDATE Reputation SET point = point + ?", (data.point * (float64(data.level) / 5)))
		if err != nil {
			log.Panicf("本地数据库错误: %s\n", err)
			return
		}
	} else {
		_, err := db.Exec("INSERT INTO Reputation (player_uuid, point) values(?,?)", data.player_uuid, (data.point * (float64(data.level) / 5)))
		if err != nil {
			log.Panicf("本地数据库错误: %s\n", err)
			return
		}
	}

	return
}

// resetReputation 重置表Reputation
func resetReputation() {
	_, err := db.Exec("delete from Reputation")
	if err != nil {
		log.Panicf("本地数据库错误: %s\n", err)
		return
	}
}

// serverList 吐出数据库Server表中的部分内容
func reportList(c chan ReportList) error {
	rows, err := db.Query("SELECT * FROM Reputation")
	if err != nil {
		return errors.New("本地数据库错误: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var data ReportList
		err := rows.Scan(&data.player_uuid, &data.point)
		if err != nil {
			return errors.New("本地数据库错误: " + err.Error())
		}
		c <- data
	}
	close(c)
	return nil
}
