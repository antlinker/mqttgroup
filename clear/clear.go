package clear

import (
	"github.com/antlinker/alog"
	"github.com/antlinker/mqttgroup/config"
	"gopkg.in/mgo.v2"
)

type Config struct {
	Gen      bool
	Pub      bool
	MongoUrl string
}

func Clear(cfg Config) {
	clr := &clear{
		lg: alog.NewALog(),
	}
	clr.lg.SetLogTag("CLEAR")
	session, err := mgo.Dial(cfg.MongoUrl)
	if err != nil {
		clr.lg.ErrorC("连接数据库出现异常：", err)
		return
	}
	clr.session = session
	clr.database = session.DB(config.Database)
	if cfg.Gen {
		err = clr.clearGroup()
		if err != nil {
			clr.lg.ErrorC("清除群组及组成员出现异常：", err)
			return
		}
	}
	if cfg.Pub {
		err = clr.clearPacket()
		if err != nil {
			clr.lg.ErrorC("清除包数据出现异常：", err)
			return
		}
	}
	clr.lg.InfoC("数据清除完成")
}

type clear struct {
	lg       *alog.ALog
	session  *mgo.Session
	database *mgo.Database
}

func (c *clear) clearGroup() error {
	c.lg.InfoC("开始清除群组及组成员...")
	_, err := c.database.C(config.CUser).RemoveAll(nil)
	if err != nil {
		return err
	}
	_, err = c.database.C(config.CGroup).RemoveAll(nil)
	if err != nil {
		return err
	}
	c.lg.InfoC("群组及组成员清除完成")
	return nil
}

func (c *clear) clearPacket() error {
	c.lg.InfoC("开始清除包数据...")
	_, err := c.database.C(config.CPacket).RemoveAll(nil)
	if err != nil {
		return err
	}
	c.lg.InfoC("包数据清除完成")
	return nil
}
