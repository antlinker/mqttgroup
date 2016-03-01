package collect

import (
	"fmt"

	"gopkg.in/mgo.v2/bson"

	"github.com/antlinker/go-bucket"

	"github.com/antlinker/alog"
	"github.com/antlinker/mqttgroup/config"

	"gopkg.in/mgo.v2"
)

// ExecCollect 执行汇总
func ExecCollect(cfg Config) {
	clt := &Collect{
		cfg:                cfg,
		lg:                 alog.NewALog(),
		chCollectPacket:    make(chan config.CollectPacket, 1),
		groupData:          make(map[string]int),
		collectPacketStore: bucket.NewBucketGroup(200, 100),
	}
	clt.lg.SetLogTag("COLLECT")
	session, err := mgo.Dial(clt.cfg.MongoUrl)
	if err != nil {
		clt.lg.Errorf("数据库连接发生异常:%s", err.Error())
		return
	}
	clt.session = session
	err = clt.Remove()
	if err != nil {
		clt.lg.Errorf("清除数据发生异常:%v", err)
		return
	}
	err = clt.Init()
	if err != nil {
		clt.lg.Errorf("数据初始化发生异常:%v", err)
		return
	}
	err = clt.StartCollect()
	if err != nil {
		clt.lg.Errorf("执行数据汇总发生异常:%v", err)
		return
	}
	err = clt.Statistics()
	if err != nil {
		clt.lg.Errorf("执行结果统计时发生异常:%v", err)
		return
	}
	clt.lg.Info("执行完成.")
}

// Collect 汇总发布的包数据
type Collect struct {
	cfg                Config
	lg                 *alog.ALog
	session            *mgo.Session
	chCollectPacket    chan config.CollectPacket
	groupData          map[string]int
	collectPacketStore bucket.BucketGroup
}

func (c *Collect) DB() *mgo.Database {
	return c.session.Clone().DB(config.Database)
}

func (c *Collect) Remove() error {
	c.lg.Info("开始清除汇总数据...")
	info, err := c.DB().C(config.CCollect).RemoveAll(nil)
	if err != nil {
		return err
	}
	c.lg.Info("数据清除完成,共清理数据条数：", info.Removed)
	return nil
}

func (c *Collect) Init() error {
	c.lg.Info("开始执行数据初始化...")
	var user config.User
	iter := c.DB().C(config.CUser).Find(nil).Iter()
	for iter.Next(&user) {
		for j := 0; j < len(user.Groups); j++ {
			gid := user.Groups[j]
			group, ok := c.groupData[gid]
			if ok {
				c.groupData[gid] = group + 1
				continue
			}
			c.groupData[gid] = 1
		}
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("初始化组用户数据发生异常：%s", err.Error())
	}
	c.collectStore()
	c.lg.Info("执行数据初始化完成.")
	return nil
}

func (c *Collect) collectStore() {
	collectBucket, err := c.collectPacketStore.Open()
	if err != nil {
		c.lg.Errorf("Open collect packet store error:%v", err)
		return
	}
	go func() {
		db := c.DB()
		for cBucket := range collectBucket {
			vals, _ := cBucket.ToSlice()
			err := db.C(config.CCollect).Insert(vals...)
			if err != nil {
				c.lg.Errorf("Store collect packet error:%v", err)
			}
		}
	}()
}

// StartCollect 开始汇总数据
func (c *Collect) StartCollect() error {
	c.lg.Info("开始执行数据汇总...")
	var sendPacket config.SendPacket
	receiveDB := c.DB()
	iter := c.DB().C(config.CPacket).Find(nil).Sort("time").Select(bson.M{"_id": 0}).Iter()
	for iter.Next(&sendPacket) {
		collectPacket := config.CollectPacket{
			PacketID:      sendPacket.SendID,
			PreReceiveNum: int64(c.groupData[sendPacket.ToGroup]),
		}
		var receivePackets []config.ReceivePacket
		err := receiveDB.C(config.CReceivePacket).Find(bson.M{"sid": sendPacket.SendID}).Select(bson.M{"_id": 0}).All(&receivePackets)
		if err != nil {
			c.lg.Errorf("Get receive packet data error:%v", err)
			continue
		}
		collectPacket.ReceiveNum = int64(len(receivePackets))
		var (
			maxConsume, minConsume, sumConsume float64
		)
		for _, receive := range receivePackets {
			consumeTime := receive.ReceiveTime.Sub(sendPacket.SendTime).Seconds()
			sumConsume += consumeTime
			if minConsume == 0 {
				minConsume = consumeTime
			}
			if consumeTime > maxConsume {
				maxConsume = consumeTime
			}
		}
		collectPacket.MaxConsume = maxConsume
		collectPacket.MinConsume = minConsume
		if collectPacket.ReceiveNum > 0 {
			collectPacket.AvgConsume = sumConsume / float64(collectPacket.ReceiveNum)
		}
		c.collectPacketStore.Push(collectPacket)
	}
	c.collectPacketStore.Close()
	if err := iter.Close(); err != nil {
		return err
	}
	c.lg.Info("数据存储写入完成.")
	return nil
}

func (c *Collect) insertData(docs ...interface{}) {
	err := c.DB().C(config.CCollect).Insert(docs...)
	if err != nil {
		c.lg.Errorf("执行汇总数据插入时发生异常!")
	}
}

// Statistics 统计结果数据
func (c *Collect) Statistics() error {
	c.lg.Info("开始执行汇总数据统计...")
	var (
		publishNum     int64
		preReceiveNum  int64
		receiveNum     int64
		rAvgSumConsume float64
		lossRate       float64
		rAvgConsume    float64
	)
	var collectPacket config.CollectPacket
	iter := c.DB().C(config.CCollect).Find(nil).Iter()
	for iter.Next(&collectPacket) {
		publishNum += 1
		preReceiveNum += collectPacket.PreReceiveNum
		receiveNum += collectPacket.ReceiveNum
		rAvgSumConsume += collectPacket.AvgConsume
	}
	if err := iter.Close(); err != nil {
		return err
	}
	output := `
统计结果如下：
连接数量        %d
总发包量        %d
总接包量        %d
总丢包率        %.3f%%
接包平均耗时    %.3fs
	`
	lossRate = (1 - float64(receiveNum)/float64(preReceiveNum)) * 100
	if receiveNum > 0 {
		rAvgConsume = rAvgSumConsume / float64(publishNum)
	}
	c.lg.Infof(output, c.clientCount(), publishNum, receiveNum, lossRate, rAvgConsume)
	return nil
}

func (c *Collect) clientCount() int {
	count, err := c.DB().C(config.CUser).Find(nil).Count()
	if err != nil {
		return 0
	}
	return count
}
