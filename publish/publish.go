package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antlinker/go-mqtt/client"

	"github.com/antlinker/alog"
	"github.com/antlinker/go-bucket"
	//MQTT "github.com/antlinker/mqtt-cli"
	"gopkg.in/mgo.v2"

	"github.com/antlinker/go-cmap"
	"github.com/antlinker/mqttgroup/config"
)

// Pub 执行Publish操作
func Pub(cfg *Config) {
	pub := &Publish{
		cfg:                cfg,
		lg:                 alog.NewALog(),
		lgData:             alog.NewALog(),
		groupData:          make(map[string]int),
		clients:            cmap.NewConcurrencyMap(),
		execComplete:       make(chan bool, 1),
		end:                make(chan bool, 1),
		startTime:          time.Now(),
		sendPacketStore:    bucket.NewBucketGroup(cfg.SendPacketStoreNum, cfg.SendPacketBucketNum),
		receivePacketStore: bucket.NewBucketGroup(cfg.ReceivePacketStoreNum, cfg.ReceivePacketBucketNum),
	}
	pub.lg.SetLogTag("PUBLISH")
	pub.lgData.SetLogTag("PUBLISH_DATA")
	session, err := mgo.Dial(cfg.MongoUrl)
	if err != nil {
		pub.lg.Errorf("数据库连接发生异常:%s", err.Error())
		return
	}
	pub.session = session
	err = pub.Init()
	if err != nil {
		pub.lg.Error(err)
		return
	}
	pub.ExecPublish()
	<-pub.end
	pub.handleEnd()
}

type Publish struct {
	cfg                *Config
	lg                 *alog.ALog
	lgData             *alog.ALog
	session            *mgo.Session
	userData           []config.User
	groupData          map[string]int
	clients            cmap.ConcurrencyMap
	publishID          int64
	execNum            int64
	execComplete       chan bool
	publishTotalNum    int64
	publishNum         int64
	prePublishNum      int64
	maxPublishNum      int64
	arrPublishNum      []int64
	receiveTotalNum    int64
	receiveNum         int64
	preReceiveNum      int64
	maxReceiveNum      int64
	arrReceiveNum      []int64
	end                chan bool
	startTime          time.Time
	publishPacketTime  time.Time
	sendPacketStore    bucket.BucketGroup
	receivePacketStore bucket.BucketGroup

	subscribeNum      int64
	subscribeTotalNum int64
}

func (p *Publish) DB() *mgo.Database {
	return p.session.Clone().DB(config.Database)
}

func (p *Publish) Init() error {
	err := p.initData()
	if err != nil {
		return err
	}
	err = p.initConnection()
	if err != nil {
		return err
	}
	p.sendStore()
	p.receiveStore()
	calcTicker := time.NewTicker(time.Second * 1)
	go p.calcMaxAndMinPacketNum(calcTicker)
	prTicker := time.NewTicker(time.Duration(p.cfg.Interval) * time.Second)
	go p.pubAndRecOutput(prTicker)
	go p.checkComplete()
	p.lg.InfoC("初始化操作完成.")
	p.publishPacketTime = time.Now()
	return nil
}

func (p *Publish) initData() error {
	p.lg.InfoC("开始组成员数据初始化...")
	err := p.DB().C(config.CUser).Find(nil).All(&p.userData)
	if err != nil {
		return fmt.Errorf("Init group user data error:%s", err.Error())
	}
	for i, l := 0, len(p.userData); i < l; i++ {
		user := p.userData[i]
		for j := 0; j < len(user.Groups); j++ {
			gid := user.Groups[j]
			group, ok := p.groupData[gid]
			if ok {
				p.groupData[gid] = group + 1
				continue
			}
			p.groupData[gid] = 1
		}
	}
	p.lg.InfoC("组成员初始化完成.")
	return nil
}

func (p *Publish) initConnection() error {
	p.lg.InfoC("开始组成员建立MQTT数据连接初始化...")
	ul := len(p.userData)
	var wgroup sync.WaitGroup
	wgroup.Add(ul)
	for i := 0; i < ul; i++ {
		go func(user config.User) {
			defer wgroup.Done()
			clientID := user.UserID
			opts := client.MqttOption{
				Addr:               fmt.Sprintf("%s://%s", p.cfg.Network, p.cfg.Address),
				Clientid:           clientID,
				ReconnTimeInterval: 1,
			}
			if name, pwd := p.cfg.UserName, p.cfg.Password; name != "" && pwd != "" {
				opts.UserName = name
				opts.Password = pwd
			}
			if v := p.cfg.KeepAlive; v > 0 {
				opts.KeepAlive = uint16(v)
				//opts.SetKeepAlive(time.Duration(v) * time.Second)
			}
			if v := p.cfg.CleanSession; v {
				//opts.SetCleanSession(v)
				opts.CleanSession = v
			}
			clientHandle := NewHandleConnect(clientID, p)
			// opts.SetConnectionLostHandler(func(cli *MQTT.Client, err error) {
			// 	clientHandle.ErrorHandle(err)
			// })
			cli, _ := client.CreateClient(opts)
			cli.AddPacketListener(clientHandle)
			cli.AddSubListener(clientHandle)
			cli.AddPubListener(clientHandle)
			cli.AddDisConnListener(clientHandle)
			cli.AddRecvPubListener(clientHandle)
		LB_RECONNECT:

			err := cli.Connect()
			if err != nil {
				p.lg.Debug("建立连接失败:", err)
				time.Sleep(time.Millisecond * 100)
				goto LB_RECONNECT
			}
			subfilters := make([]client.SubFilter, len(user.Groups))
			//subTopics := make(map[string]byte)
			for j := 0; j < len(user.Groups); j++ {
				subfilters[j] = client.CreateSubFilter("G/"+user.Groups[j], client.QoS(p.cfg.Qos))
			}
		LB_SUBSCRIBE:
			atomic.AddInt64(&p.subscribeTotalNum, 1)
			mp, err := cli.Subscribes(subfilters...)
			if err != nil {
				cli.Disconnect()
				p.lg.Debug("重新连接:", err)
				goto LB_RECONNECT
			}
			mp.Wait()
			if mp.Err() != nil {
				p.lg.Debug("重新订阅:", err)
				goto LB_SUBSCRIBE
			}
			p.clients.Set(clientID, cli)
		}(p.userData[i])
	}
	wgroup.Wait()
	p.lg.InfoC("组成员建立MQTT数据连接完成.")
	return nil
}

func (p *Publish) sendStore() {
	bGroup, err := p.sendPacketStore.Open()
	if err != nil {
		p.lg.Errorf("Send store error:%v", err)
		return
	}
	go func() {
		db := p.DB()
		for cBucket := range bGroup {
			if cBucket.Len() == 0 {
				continue
			}
		LB_SENDPACKETSTORE:
			sVals, _ := cBucket.ToSlice()
			err := db.C(config.CPacket).Insert(sVals...)
			if err != nil {
				p.lg.Errorf("Send packet store error!")
				time.Sleep(time.Millisecond * 1)
				goto LB_SENDPACKETSTORE
			}
		}
	}()
}

func (p *Publish) receiveStore() {
	bGroup, err := p.receivePacketStore.Open()
	if err != nil {
		p.lg.Errorf("Receive store error:%v", err)
		return
	}
	go func() {
		db := p.DB()
		for cBucket := range bGroup {
			if cBucket.Len() == 0 {
				continue
			}
		LB_RECEIVEPACKETSTORE:
			sVals, _ := cBucket.ToSlice()
			err := db.C(config.CReceivePacket).Insert(sVals...)
			if err != nil {
				p.lg.Errorf("Receive packet store error:%v", err)
				time.Sleep(time.Millisecond * 1)
				goto LB_RECEIVEPACKETSTORE
			}
		}
	}()
}

func (p *Publish) calcMaxAndMinPacketNum(ticker *time.Ticker) {
	for range ticker.C {
		pubNum := p.publishNum
		recNum := p.receiveNum
		pNum := pubNum - p.prePublishNum
		rNum := recNum - p.preReceiveNum
		if pNum > p.maxPublishNum {
			p.maxPublishNum = pNum
		}
		p.arrPublishNum = append(p.arrPublishNum, pNum)
		if rNum > p.maxReceiveNum {
			p.maxReceiveNum = rNum
		}
		p.arrReceiveNum = append(p.arrReceiveNum, rNum)
		p.prePublishNum = pubNum
		p.preReceiveNum = recNum
	}
}

func (p *Publish) pubAndRecOutput(ticker *time.Ticker) {
	for ct := range ticker.C {
		totalSecond := float64(ct.Sub(p.startTime)) / float64(time.Second)
		packetSecond := (float64(ct.Sub(p.publishPacketTime)) - float64(time.Duration(p.cfg.Interval)*time.Second)) / float64(time.Second)
		var psNum int64
		arrPNum := p.arrPublishNum
		for i := 0; i < len(arrPNum); i++ {
			psNum += arrPNum[i]
		}
		avgPNum := psNum / int64(len(arrPNum))
		var rsNum int64
		arrRNum := p.arrReceiveNum
		for i := 0; i < len(arrRNum); i++ {
			rsNum += arrRNum[i]
		}
		avgRNum := rsNum / int64(len(arrRNum))
		clientNum := p.clients.Len()
		output := `
总耗时                      %.2fs
发包耗时                    %.2fs
执行次数                    %d
客户端数量                  %d
订阅数量                    %d
实际订阅量                  %d
应发包量                    %d
实际发包量                  %d
每秒平均的发包量            %d
每秒最大的发包量            %d
应收包量                    %d
实际收包量                  %d
每秒平均的接包量            %d
每秒最大的接包量            %d`

		p.lgData.Infof(output,
			totalSecond, packetSecond, p.execNum, clientNum, p.subscribeTotalNum, p.subscribeNum, p.publishTotalNum, p.publishNum, avgPNum, p.maxPublishNum, p.receiveTotalNum, p.receiveNum, avgRNum, p.maxReceiveNum)
	}
}

func (p *Publish) checkComplete() {
	<-p.execComplete
	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		for range ticker.C {
			if p.publishNum == p.publishTotalNum {
				p.sendPacketStore.Close()
				ticker.Stop()
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		for range ticker.C {
			if p.receiveNum == p.receiveTotalNum {
				ticker.Stop()
				p.end <- true
			}
		}
	}()
}

func (p *Publish) ExecPublish() {
	time.AfterFunc(time.Duration(p.cfg.Interval)*time.Second, func() {
		if p.execNum == int64(p.cfg.ExecNum) {
			p.lg.InfoC("发布已执行完成，等待接收订阅...")
			p.disconnect()
			p.execComplete <- true
			return
		}
		atomic.AddInt64(&p.execNum, 1)
		p.publish()
		p.ExecPublish()
	})
}

func (p *Publish) disconnect() {
	if v := p.cfg.DisconnectScale; v > 100 || v <= 0 {
		return
	}
	clientCount := p.clients.Len()
	disCount := int(float32(clientCount) * (float32(p.cfg.DisconnectScale) / 100))
	for _, v := range p.clients.ToMap() {
		vc := v.(client.MqttClienter)
		vc.Disconnect()
		disCount--
		if disCount == 0 {
			break
		}
	}
}

func (p *Publish) publish() {
	for i, l := 0, len(p.userData); i < l; i++ {
		user := p.userData[i]
		groups := user.Groups
		for j := 0; j < len(groups); j++ {
			p.publishTotalNum++
			p.receiveTotalNum += int64(p.groupData[groups[j]])
		}
		go p.userPublish(user)
		if v := p.cfg.UserInterval; v > 0 {
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}
}

func (p *Publish) userPublish(user config.User) {
	ucli, _ := p.clients.Get(user.UserID)
	if ucli == nil {
		return
	}
	cli := ucli.(client.MqttClienter)
	for i := 0; i < len(user.Groups); i++ {
		group := user.Groups[i]
		cTime := time.Now()
		sendPacket := config.SendPacket{
			SendID:   fmt.Sprintf("%d_%d", cTime.Unix(), atomic.AddInt64(&p.publishID, 1)),
			SendUser: user.UserID,
			ToGroup:  group,
			SendTime: cTime,
		}
		if p.cfg.IsStore {
			p.sendPacketStore.Push(sendPacket)
			// err := p.database.C(config.CPacket).Insert(sendPacket)
			// if err != nil {
			// 	p.lg.Errorf("Publish store error:%s", err.Error())
			// }
		}
		bufData, err := json.Marshal(sendPacket)
		if err != nil {
			p.lg.Errorf("Publish json marshal error:%s", err.Error())
			continue
		}
		topic := "G/" + group
		cli.Publish(topic, client.QoS(p.cfg.Qos), false, bufData)

		if v := p.cfg.UserGroupInterval; v > 0 {
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}
}

func (p *Publish) handleEnd() {
	p.receivePacketStore.Close()
	p.lg.InfoC("执行完成.")
	time.Sleep(time.Second * 2)
	os.Exit(0)
}
