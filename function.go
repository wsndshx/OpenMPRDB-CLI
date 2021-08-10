package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
)

// register 在中心服务器上注册本服务器
func register(server_name string) (string, error) {
	// 生成请求内容
	register := make(map[string]string)
	message, err := GenerateSignedMessage("server_name: " + server_name)

	// 读取本地公钥
	pubkey, err := ioutil.ReadFile("rsa-pub.pem")
	if err != nil {
		return "", errors.New("读取本地公钥错误: " + err.Error())
	}

	register["message"] = message
	register["public_key"] = string(pubkey)
	bytesData, _ := json.Marshal(register)

	// PUT请求 [服务器地址]/v1/server/register
	req, err := httpRequest("PUT", "application/json", "/v1/server/register", bytes.NewBuffer(bytesData))
	var data struct {
		Status string `json:"status"`
		UUID   string `json:"uuid"`
		Reason string `json:"reason"`
	}

	err = json.Unmarshal(req, &data)
	if err != nil {
		return "", errors.New("序列化错误: " + err.Error())
	}
	if data.Status == "NG" {
		return "", errors.New("中心服务器返回异常: " + data.Reason)
	}

	// 返回得到的uuid
	return data.UUID, nil
}

// newSubmit 在中心服务器上提交新玩家数据
func newSubmit(player, comment string, point int) (string, error) {
	// 生成请求数据
	message, err := GenerateSignedMessage(fmt.Sprintf("uuid: %s\r\ntimestamp: %d\r\nplayer_uuid: %s\r\npoints: %d\r\ncomment: %s", uuid.Must(uuid.NewV4(), nil).String(), time.Now().Unix(), player, point, comment))

	if err != nil {
		return "", err
	}

	// PUT请求: [API服务器地址]/v1/submit/new
	req, err := httpRequest("PUT", "text/plain", "/v1/submit/new", bytes.NewBufferString(message))
	var data struct {
		Status string `json:"status"`
		UUID   string `json:"uuid"`
		Reason string `json:"reason"`
	}

	// 序列化
	err = json.Unmarshal(req, &data)
	if err != nil {
		return "", errors.New("序列化错误: " + err.Error())
	}
	if data.Status == "NG" {
		return "", errors.New("中心服务器返回异常: " + data.Reason)
	}

	return data.UUID, nil
}

// deleteSubmit 删除过去提交到服务器上的一条记录
func deleteSubmit(uuid, comment string) error {
	// 生成请求数据
	message, err := GenerateSignedMessage(fmt.Sprintf("timestamp: %d\r\ncomment: %s", time.Now().Unix(), comment))
	if err != nil {
		return err
	}

	// PUT请求: [API服务器地址]/v1/submit/uuid/<submit_uuid>
	req, err := httpRequest("DELETE", "text/plain", "/v1/submit/uuid/"+uuid, bytes.NewBufferString(message))
	if err != nil {
		return err
	}
	var data struct {
		Status string `json:"status"`
		UUID   string `json:"uuid"`
		Reason string `json:"reason"`
	}

	// 序列化
	err = json.Unmarshal(req, &data)
	if err != nil {
		return errors.New("序列化错误: " + err.Error())
	}
	if data.Status == "NG" {
		return errors.New("中心服务器返回异常: " + data.Reason)
	}
	return nil
}

// getServerData 获取指定服务器的数据
func getServerData(uuid, pubkey string) error {
	// GET请求: [API服务器地址]/v1/submit/server/<server_uuid>
	resp, err := http.Get(serverAddress + "/v1/submit/server/" + uuid)
	if err != nil {
		return errors.New("发送请求错误: " + err.Error())
	}
	defer resp.Body.Close()

	// 读取返回值
	pageBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("读取返回值错误: " + err.Error())
	}
	type submits struct {
		ID          int    `json:"id"`
		UUID        string `json:"uuid"`
		Server_uuid string `json:"server_uuid"`
		Content     string `json:"content"`
	}
	type data struct {
		Status  string    `json:"status"`
		Reason  string    `json:"reason"`
		Submits []submits `json:"submits"`
	}
	var Data data

	// 序列化
	err = json.Unmarshal(pageBytes, &Data)
	if err != nil {
		return errors.New("序列化错误: " + err.Error())
	}
	if Data.Status == "NG" {
		return errors.New("中心服务器返回异常: " + Data.Reason)
	}
	return nil
}

// submissionList 获取提交列表
func submissionList() error {
	c := make(chan string)
	fmt.Println("\t\t操作uuid\t\t|\t\t玩家uuid\t\t|  评分\t|理由")
	go subList(c)
	for i := range c {
		fmt.Println(i)
	}
	log.Println("已到达最底端")
	return nil
}

// trustServer 信任某个服务器
func trustServer(uuid string, name string, pubkey_path string) error {
	// 将信息存入数据库
	err := insertServer(uuid, name, pubkey_path)
	if err != nil {
		return err
	}
	return nil
}

// listServers 列出服务器列表(已信任)
func listServers() error {
	c := make(chan string)
	fmt.Println("\t\t服务器uuid\t\t|名称")
	go serverList(c)
	for i := range c {
		fmt.Println(i)
	}
	log.Println("已到达最底端")
	return nil
}
