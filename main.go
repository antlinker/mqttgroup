package main

import (
	"os"

	"github.com/antlinker/mqttgroup/publish"

	"gopkg.in/alog.v1"

	"github.com/antlinker/mqttgroup/clear"

	"github.com/antlinker/mqttgroup/generate"

	"github.com/codegangsta/cli"
)

func main() {
	alog.RegisterAlog("conf/log.yaml")
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
				Value: 200,
				Usage: "组成员数量限制",
			},
			cli.IntFlag{
				Name:  "clientnum, cn",
				Value: 1000,
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
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "ExecNum, en",
				Value: 10,
				Usage: "执行次数",
			},
			cli.IntFlag{
				Name:  "Interval, i",
				Value: 5,
				Usage: "发布间隔(单位:秒)",
			},
			cli.IntFlag{
				Name:  "UserInterval, ui",
				Value: 1,
				Usage: "组成员发包间隔（单位:毫秒）",
			},
			cli.IntFlag{
				Name:  "UserGroupInterval, ugi",
				Value: 0,
				Usage: "组成员针对组的发包间隔（单位:毫秒）",
			},
			cli.IntFlag{
				Name:  "AutoReconnect, ar",
				Value: 1,
				Usage: "客户端断开连接后执行自动重连(默认为1，0表示不重连)",
			},
			cli.IntFlag{
				Name:  "DisconnectScale, ds",
				Value: 0,
				Usage: "发送完成之后，需要断开客户端的比例",
			},
			cli.BoolFlag{
				Name:  "IsStore, s",
				Usage: "是否执行持久化存储",
			},
			cli.StringFlag{
				Name:  "Network, net",
				Value: "tcp",
				Usage: "MQTT Network",
			},
			cli.StringFlag{
				Name:  "Address, addr",
				Value: "127.0.0.1:1883",
				Usage: "MQTT Address",
			},
			cli.StringFlag{
				Name:  "UserName, name",
				Value: "",
				Usage: "MQTT UserName",
			},
			cli.StringFlag{
				Name:  "Password, pwd",
				Value: "",
				Usage: "MQTT Password",
			},
			cli.IntFlag{
				Name:  "QOS, qos",
				Value: 1,
				Usage: "MQTT QOS",
			},
			cli.IntFlag{
				Name:  "KeepAlive, alive",
				Value: 60,
				Usage: "MQTT KeepAlive",
			},
			cli.BoolFlag{
				Name:  "CleanSession, cs",
				Usage: "MQTT CleanSession",
			},
			cli.StringFlag{
				Name:  "mongo, mgo",
				Value: "mongodb://127.0.0.1:27017",
				Usage: "MongoDB连接url",
			},
		},
		Action: func(ctx *cli.Context) {
			cfg := &publish.Config{
				ExecNum:           ctx.Int("ExecNum"),
				Interval:          ctx.Int("Interval"),
				UserInterval:      ctx.Int("UserInterval"),
				UserGroupInterval: ctx.Int("UserGroupInterval"),
				IsStore:           ctx.Bool("IsStore"),
				Network:           ctx.String("Network"),
				Address:           ctx.String("Address"),
				Qos:               byte(ctx.Int("QOS")),
				UserName:          ctx.String("UserName"),
				Password:          ctx.String("Password"),
				CleanSession:      ctx.Bool("CleanSession"),
				KeepAlive:         ctx.Int("KeepAlive"),
				MongoUrl:          ctx.String("mongo"),
			}
			if ctx.Int("AutoReconnect") == 1 {
				cfg.AutoReconnect = true
			}
			publish.Pub(cfg)
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
