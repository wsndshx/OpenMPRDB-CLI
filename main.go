package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"log"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var SqlPath string = "./OpenMPRDB.db"
var bar *progressbar.ProgressBar

func init() {

}

func main() {
	app := &cli.App{
		Name:  "OpenMPRDB-CLI",
		Usage: "一个简陋的客户端",
		Commands: []*cli.Command{
			{
				Name:  "update",
				Usage: "更新信誉信息",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "export",
						Usage: "更新完成后将结果导出到文件中",
						Value: "",
					},
					&cli.Float64Flag{
						Name:  "less",
						Usage: "只输出小于某值的数据",
						Value: -0,
					},
				},
				Action: func(c *cli.Context) error {
					// 显示一个进度条, 防止时间过长
					bar = progressbar.Default(1)
					bar.ChangeMax(0)
					// 生成数据
					generateReport()

					// 输出一下
					c1 := make(chan ReportList)
					c2 := make(chan string)
					var export bool = false
					if c.String("export") != "" {
						export = true
						go exportBanList(c.String("export"), c2)
					}
					fmt.Println("\t\t玩家uuid\t\t|评分")
					go reportList(c1)
					if c.Float64("less") != -0 {
						for i := range c1 {
							if i.point <= c.Float64("less") {
								fmt.Println(fmt.Sprintf("%s\t|%.1f", i.player_uuid, i.point))
								if export {
									c2 <- i.player_uuid
								}
							}
						}
					} else {
						for i := range c1 {
							fmt.Println(fmt.Sprintf("%s\t|%.1f", i.player_uuid, i.point))
							if export {
								c2 <- i.player_uuid
							}
						}
					}
					close(c2)
					time.Sleep(1)
					log.Println("已到达最底端")
					return nil
				},
			},
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
					&cli.IntFlag{
						Name:     "level",
						Usage:    "Trust level. (1 ~ 5)",
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
					err := trustServer(c.String("uuid"), c.String("name"), c.String("pubkey"), c.Int("level"))
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
					&cli.StringFlag{
						Name:     "remote",
						Value:    "OpenMPRDB-CLI_test",
						Usage:    "The address of the server.",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					// 在中心服务器上注册本客户端(服务器)
					server_uuid, err := register(c.String("server_name"), c.String("remote"))
					if err != nil {
						return err
					}

					// 在数据库中存储数据
					err = registerServer(c.String("server_name"), server_uuid, c.String("remote"))
					if err != nil {
						return err
					}

					log.Printf("服务器%s[%s]注册成功", c.String("server_name"), server_uuid)
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
					&cli.Float64Flag{
						Name:     "point",
						Value:    0,
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
					uuid, err := newSubmit(c.String("player"), c.String("comment"), c.Float64("point"))
					if err != nil {
						return err
					}

					// 在数据库中存储提交数据
					err = newSubmission(uuid, c.String("player"), c.String("comment"), c.Float64("point"))
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
func httpRequest(method, Type, serverAddress, API string, data io.Reader) ([]byte, error) {
	req, _ := http.NewRequest(method, serverAddress+API, data)
	req.Header.Add("Content-Type", Type)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("发送请求错误: " + err.Error())
	}
	defer res.Body.Close()

	//读取返回的内容
	pageBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("读取返回值错误: " + err.Error())
	}
	return pageBytes, nil
}
