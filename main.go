package main

import (
	"fmt"
	"os"

	"github.com/ant-testing/mqttgroup/clear"

	"github.com/ant-testing/mqttgroup/generate"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "mqttgroup"
	app.Author = "Lyric"
	app.Version = "0.1.0"
	app.Usage = "MQTT群组聊天测试"
	app.Commands = append(app.Commands, cli.Command{
		Name:    "generate",
		Aliases: []string{"gen"},
		Usage:   "生成群组并且分配群成员",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "groupnum, gn",
				Value: 200,
				Usage: "群组数量",
			},
			cli.IntFlag{
				Name:  "grouplimit, gcln",
				Value: 2000,
				Usage: "组成员数量限制",
			},
			cli.IntFlag{
				Name:  "clientnum, cn",
				Value: 10000,
				Usage: "组成员数量",
			},
			cli.IntFlag{
				Name:  "clientlimit, cgln",
				Value: 50,
				Usage: "组成员所拥有的组数量限制",
			},
			cli.StringFlag{
				Name:  "mongo, mgo",
				Value: "mongodb://127.0.0.1:27017",
				Usage: "MongoDB连接url",
			},
		},
		Action: func(ctx *cli.Context) {
			cfg := generate.Config{
				GroupNum:            ctx.Int("groupnum"),
				GroupClientLimitNum: ctx.Int("grouplimit"),
				ClientNum:           ctx.Int("clientnum"),
				ClientGroupLimitNum: ctx.Int("clientlimit"),
				MongoUrl:            ctx.String("mongo"),
			}
			generate.Gen(cfg)
		},
	})
	app.Commands = append(app.Commands, cli.Command{
		Name:    "publish",
		Aliases: []string{"pub"},
		Usage:   "发布消息",
		Action: func(ctx *cli.Context) {
			fmt.Println("===> Publish")
		},
	})
	app.Commands = append(app.Commands, cli.Command{
		Name:    "clear",
		Aliases: []string{"c"},
		Usage:   "清除数据",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "generate, gen",
				Usage: "清除群组基础数据",
			},
			cli.BoolFlag{
				Name:  "publish, pub",
				Usage: "清除publish包数据",
			},
			cli.StringFlag{
				Name:  "mongo, mgo",
				Value: "mongodb://127.0.0.1:27017",
				Usage: "MongoDB连接url",
			},
		},
		Action: func(ctx *cli.Context) {
			cfg := clear.Config{
				Gen:      ctx.Bool("generate"),
				Pub:      ctx.Bool("publish"),
				MongoUrl: ctx.String("mongo"),
			}
			clear.Clear(cfg)
		},
	})
	app.Run(os.Args)
}
