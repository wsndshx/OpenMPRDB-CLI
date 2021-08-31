package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	uuid "github.com/satori/go.uuid"
)

// register 在中心服务器上注册本服务器
func register(server_name string) (string, error) {
	// 生成请求内容
	register := make(map[string]string)
	message, err := SignatureData("server_name: " + server_name)

	// 读取本地公钥
	var pubkey string
	err = db.QueryRow("SELECT public_key FROM Config").Scan(&pubkey)
	if err != nil {
		return "", errors.New("无法读取本地私钥: " + err.Error())
	}

	register["message"] = message
	register["public_key"] = pubkey
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
func newSubmit(player, comment string, point float64) (string, error) {
	// 生成请求数据
	message, err := SignatureData(fmt.Sprintf("uuid: %s\r\ntimestamp: %d\r\nplayer_uuid: %s\r\npoints: %.1f\r\ncomment: %s", uuid.Must(uuid.NewV4(), nil).String(), time.Now().Unix(), player, point, comment))

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
	message, err := SignatureData(fmt.Sprintf("timestamp: %d\r\ncomment: %s", time.Now().Unix(), comment))
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
func getServerData(uuid, pubkey string, level int, c chan SubList) {
	// GET请求: [API服务器地址]/v1/submit/server/<server_uuid>
	resp, err := http.Get(serverAddress + "/v1/submit/server/" + uuid)
	if err != nil {
		log.Panicf("发送请求错误: %s\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取返回值
	pageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("读取返回值错误: %s\n", err)
		return
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
		log.Panicf("序列化错误: %s\n", err)
		return
	}
	if Data.Status == "NG" {
		log.Panicf("中心服务器返回异常: %s\n", Data.Reason)
		return
	}
	// log.Printf("正在加载服务器[%s]的数据\n", uuid)
	for _, s := range Data.Submits {
		// 对数据进行验签
		verifiedPlainText, err := helper.VerifyCleartextMessageArmored(pubkey, s.Content, crypto.GetUnixTime())
		if err != nil {
			log.Panicf("消息验签失败: %s\n", err)
			return
		}
		// 提取数据
		flysnowRegexp := regexp.MustCompile(`^uuid: (.{36})\ntimestamp: ([0-9]*)\nplayer_uuid: (.{36})\npoints: (.{1,4})\ncomment: (.*)$`)
		params := flysnowRegexp.FindStringSubmatch(verifiedPlainText)
		point, err := strconv.ParseFloat(params[4], 32)
		if err != nil {
			log.Panicf("无法获取评分: %s\n", err)
			return
		}
		// 存储数据
		data := SubList{
			uuid:        params[1],
			player_uuid: params[3],
			comment:     params[5],
			point:       float64(point),
			level:       level,
		}
		c <- data
	}
	// log.Printf("服务器[%s]的数据加载完成\n", uuid)

	close(c)

	return
}

// submissionList 获取提交列表
func submissionList() error {
	c := make(chan SubList)
	fmt.Println("\t\t操作uuid\t\t|\t\t玩家uuid\t\t|  评分\t|理由")
	go subList(c)
	for i := range c {
		fmt.Println(fmt.Sprintf("%s\t|%s\t|   %.1f\t|%s", i.uuid, i.player_uuid, i.point, i.comment))
	}
	log.Println("已到达最底端")
	return nil
}

// trustServer 信任某个服务器
func trustServer(uuid string, name string, pubkey_path string, level int) error {
	// 将信息存入数据库
	err := insertServer(uuid, name, pubkey_path, level)
	if err != nil {
		return err
	}
	return nil
}

// listServers 列出服务器列表(已信任)
func listServers() error {
	c := make(chan ServerList)
	fmt.Println("\t\t服务器uuid\t\t|名称")
	go serverList(c)
	for i := range c {
		fmt.Println(fmt.Sprintf("%s\t|%s", i.uuid, i.name))
	}
	log.Println("已到达最底端")
	return nil
}

// generateReport 生成信誉报告
func generateReport() {
	// 插入队列
	c4 := make(chan SubList, 65536)

	// 清空表Reputation
	resetReputation()

	// 开始生成数据

	// 读取表Submission
	c1 := make(chan SubList, 2048)
	go subList(c1)
	// 插入数据
	for i := range c1 {
		bar.ChangeMax(bar.GetMax() + 1)
		c4 <- i
	}

	//读取表Serves
	c2 := make(chan ServerList)
	go serverList(c2)
	var server []ServerList
	for a := range c2 {
		server = append(server, a)
	}

	miao := 0

	for _, sl := range server {
		miao++
		c3 := make(chan SubList)
		// 获取服务器数据
		go getServerData(sl.uuid, sl.pubkey, sl.level, c3)
		// 加入队列
		go func() {
			for b := range c3 {
				bar.ChangeMax(bar.GetMax() + 1)
				c4 <- b
			}
			miao--
			if miao == 0 {
				close(c4)
			}
		}()
	}

	// 写入数据
	for b := range c4 {
		addReputation(b)
		bar.Add(1)
	}
	bar.Add(1)
}

// 导出banlist到指定文件
func exportBanList(path string, ch chan string) {
	type ban struct {
		UUID    string `json:"uuid"`
		Created string `json:"created"`
		Source  string `json:"source"`
		Expires string `json:"expires"`
		Reason  string `json:"reason"`
	}
	var banList []ban
	// 这里接收要写入的数据
	for uuid := range ch {
		miao := ban{
			UUID:    uuid,
			Created: time.Now().Format("2006-01-02 15:04:05 -0700"),
			Source:  "OpenMPRDB-CLI",
			Expires: "forever",
			Reason:  "OpenMPRDB-CLI Ban",
		}
		banList = append(banList, miao)
	}
	// 写入文件
	data, err := json.Marshal(banList)
	if err != nil {
		log.Panicf("反序列化错误: %s\n", err)
		return
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Panicf("文件写入错误: %s\n", err)
		return
	}
	return
}
