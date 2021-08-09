package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	req, err := httpPUT("application/json", "/v1/server/register", bytes.NewBuffer(bytesData))
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
	// message, err := GenerateSignedMessage(fmt.Sprintf("uuid: %s\ntimestamp: %d\nplayer_uuid: %s\npoints: %d\ncomment: %s\n", uuid.Must(uuid.NewV4(), nil).String(), time.Now().Unix(), player, point, comment))
	message, err := GenerateSignedMessage("miao\n123")

	fmt.Println(message)
	if err != nil {
		return "", err
	}

	// PUT请求: [API服务器地址]/v1/submit/new
	req, err := httpPUT("text/plain", "/v1/submit/new", bytes.NewBufferString(message))
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