package generate

import (
	"fmt"
	"math/rand"
	"time"

	"gopkg.in/alog.v1"

	"github.com/ant-testing/mqttgroup/config"

	"gopkg.in/mgo.v2"
)

type Generate struct {
	cfg            Config
	lg             *alog.ALog
	session        *mgo.Session
	database       *mgo.Database
	groupWeight    map[int]string
	groupClientNum map[string]int
}

// Gen 生成群组及组成员操作
func Gen(cfg Config) {
	gen := &Generate{
		cfg:            cfg,
		lg:             alog.NewALog(),
		groupWeight:    make(map[int]string),
		groupClientNum: make(map[string]int),
	}
	gen.lg.SetLogTag("GENERATE")
	session, err := mgo.Dial(cfg.MongoUrl)
	if err != nil {
		gen.lg.ErrorC("连接数据库出现异常：", err)
		return
	}
	gen.session = session
	gen.database = session.DB(config.Database)
	err = gen.GenGroup()
	if err != nil {
		gen.lg.ErrorC("生成群组出现异常：", err)
		return
	}
	err = gen.GenUser()
	if err != nil {
		gen.lg.ErrorC("生成组成员出现异常：", err)
		return
	}
	gen.lg.InfoC("群组及组成员生成完成")
}

func (g *Generate) GenGroup() error {
	g.lg.InfoC("开始生成群组...")
	groupData := make([]interface{}, g.cfg.GroupNum)
	for i := 0; i < g.cfg.GroupNum; i++ {
		var group config.Group
		group.GroupID = fmt.Sprintf("G%d_%d", time.Now().Unix(), i+1)
		group.Weight = i + 1
		groupData[i] = group
		g.groupWeight[group.Weight] = group.GroupID
	}
	err := g.database.C(config.CGroup).Insert(groupData...)
	if err != nil {
		return err
	}
	g.lg.InfoC("群组生成完成")
	return nil
}

func (g *Generate) GenUser() error {
	g.lg.InfoC("开始生成用户并为用户分配群组...")
	var maxNum, minNum, clientNum int
	for i := 0; i < g.cfg.ClientNum; i++ {
		var groupData []string
		for j := 0; j < g.cfg.ClientGroupLimitNum; j++ {
			w := g.getWeight()
			if w > g.cfg.GroupNum {
				continue
			}
			gid := g.groupWeight[w]
			var exist bool
			for k := 0; k < len(groupData); k++ {
				if groupData[k] == gid {
					exist = true
					break
				}
			}
			if !exist && g.groupClientNum[gid] < g.cfg.GroupClientLimitNum {
				groupData = append(groupData, gid)
			}
		}
		for j := 0; j < len(groupData); j++ {
			gid := groupData[j]
			if v, ok := g.groupClientNum[gid]; ok {
				g.groupClientNum[gid] = v + 1
			} else {
				g.groupClientNum[gid] = 1
			}
		}
		groupLen := len(groupData)
		if groupLen == 0 {
			continue
		}
		if minNum == 0 {
			minNum = groupLen
		}
		if groupLen > maxNum {
			maxNum = groupLen
		}
		if groupLen < minNum {
			minNum = groupLen
		}
		user := config.User{
			UserID: fmt.Sprintf("U%d_%d", time.Now().Unix(), i+1),
			Groups: groupData,
		}
		err := g.database.C(config.CUser).Insert(user)
		if err != nil {
			return err
		}
		clientNum++
	}
	g.lg.InfoCf("群组分配完成,组成员总数:%d,成员所拥有的群组数最多:%d,最少:%d", clientNum, maxNum, minNum)
	return nil
}

func (g *Generate) getWeight() int {
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := rd.Intn(g.cfg.GroupNum * 2)
	if n == 0 {
		n = g.getWeight()
	}
	return n
}
