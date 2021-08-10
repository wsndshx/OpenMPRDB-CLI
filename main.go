package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

var SqlPath string = "./OpenMPRDB.db"
var serverAddress string = "https://test.openmprdb.org"

func init() {
	// log.SetFlags(log.Ldate | log.Lshortfile)
	// 检查本地密钥是否存在
	if !Exists("rsa-priv.pem") {
		log.Println("本地密钥文件不存在, 将在默认位置生成密钥文件")
		// 初始化本地密钥
		err := initializationKey()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("密钥文件生成成功, 请勿移除公钥文件(rsa-pub.pem)和私钥文件(rsa-priv.pem), 并且妥善保管相关副本")
	}
}

func main() {
	app := &cli.App{
		Name:  "OpenMPRDB-CLI",
		Usage: "好无聊（滚滚滚）",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "列出一些东西，例如提交历史...",
				Subcommands: []*cli.Command{
					{
						Name:    "submission",
						Usage:   "List the history of the submission.",
						Aliases: []string{"sub"},
						Action: func(c *cli.Context) error {
							err := submissionList()
							if err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:  "server",
						Usage: "Make a list of trusted servers.",
						Action: func(c *cli.Context) error {
							// 这里列出server表的内容
							err := listServers()
							if err != nil {
								return err
							}
							return nil
						},
					},
				},
			},
			{
				Name:  "import",
				Usage: "Import and trust server-specific data.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "pubkey",
						Usage:    "Public key path.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "uuid",
						Usage:    "Server uuid",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "name",
						Usage: "Server name",
						Value: "Kuroneko",
					},
				},
				Action: func(c *cli.Context) error {
					// 将相应的信息存入数据库
					err := trustServer(c.String("uuid"), c.String("name"), c.String("pubkey"))
					if err != nil {
						return err
					}
					log.Printf("信任服务器 %s 成功, 已存储对应的公钥\n", c.String("uuid"))
					return nil
				},
			},
			{
				Name:  "register",
				Usage: "Register this server on the central server.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server_name",
						Value:    "OpenMPRDB-CLI_test",
						Usage:    "The name of the server.",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					// 在中心服务器上注册本客户端(服务器)
					server_uuid, err := register(c.String("server_name"))
					if err != nil {
						return err
					}

					// 在数据库中存储数据
					err = registerServer(c.String("server_name"), server_uuid)
					if err != nil {
						return err
					}

					log.Println("服务器注册成功")
					return nil
				},
			},
			{
				Name:  "new",
				Usage: "Add player popularity data.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "player",
						Value:    "a5fac3b4-ff62-4...",
						Usage:    "Specify the player's uuid.",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "point",
						Value:    -1,
						Usage:    "Specify the player's uuid.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "comment",
						Value:    "Banned for...",
						Usage:    "The reason for doing so.",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					uuid, err := newSubmit(c.String("player"), c.String("comment"), c.Int("point"))
					if err != nil {
						return err
					}

					// 在数据库中存储提交数据
					err = newSubmission(uuid, c.String("player"), c.String("comment"), c.Int("point"))
					if err != nil {
						return err
					}

					log.Printf("玩家数据提交成功, 操作uuid: %s", uuid)
					return nil
				},
			},
			{
				Name:  "delete",
				Usage: "Delete the specified past submission.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "submit",
						Value:    "a5fac3b4-ff62-4...",
						Usage:    "Specify the submitted uuid.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "comment",
						Value:    "revert ...",
						Usage:    "Delete the reason for the submission.",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					// 向中心服务器提交删除请求
					err := deleteSubmit(c.String("submit"), c.String("comment"))
					if err != nil {
						return err
					}

					// 删除本地数据库中的记录
					err = deleteSubmission(c.String("submit"))
					if err != nil {
						return err
					}

					log.Printf("提交 %s 删除成功!", c.String("submit"))
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}

//Exists 判断文件是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// httpRequest 向指定接口以指定方法发送数据, 返回得到的内容
func httpRequest(method, Type, API string, data io.Reader) ([]byte, error) {
	req, _ := http.NewRequest(method, serverAddress+API, data)
	req.Header.Add("Content-Type", Type)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("发送请求错误: " + err.Error())
	}
	defer res.Body.Close()

	//读取返回的内容
	pageBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("读取返回值错误: " + err.Error())
	}
	return pageBytes, nil
}
