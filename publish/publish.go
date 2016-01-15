package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/antlinker/go-cmap"

	"github.com/ant-testing/mqttgroup/config"
	"github.com/yosssi/gmq/mqtt/client"
	"gopkg.in/alog.v1"
	"gopkg.in/mgo.v2"
)

// Pub 执行Publish操作
func Pub(cfg *Config) {
	pub := &Publish{
		cfg:          cfg,
		lg:           alog.NewALog(),
		groupData:    make(map[string]int),
		clients:      make(map[string]*client.Client),
		disClients:   cmap.NewConcurrencyMap(),
		execComplete: make(chan bool, 1),
		end:          make(chan bool, 1),
		startTime:    time.Now(),
	}
	pub.lg.SetLogTag("PUBLISH")
	session, err := mgo.Dial(cfg.MongoUrl)
	if err != nil {
		pub.lg.Errorf("数据库连接发生异常:%s", err.Error())
		return
	}
	pub.session = session
	pub.database = session.DB(config.Database)
	err = pub.Init()
	if err != nil {
		pub.lg.Error(err)
		return
	}
	pub.ExecPublish()
	<-pub.end
	pub.lg.InfoC("执行完成.")
	os.Exit(0)
}

type Publish struct {
	cfg             *Config
	lg              *alog.ALog
	session         *mgo.Session
	database        *mgo.Database
	userData        []config.User
	groupData       map[string]int
	clients         map[string]*client.Client
	disClients      cmap.ConcurrencyMap
	publishID       int64
	execNum         int64
	execComplete    chan bool
	publishTotalNum int64
	publishNum      int64
	prePublishNum   int64
	maxPublishNum   int64
	arrPublishNum   []int64
	receiveTotalNum int64
	receiveNum      int64
	preReceiveNum   int64
	maxReceiveNum   int64
	arrReceiveNum   []int64
	end             chan bool
	startTime       time.Time
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
	p.lg.InfoC("初始化操作完成.")
	calcTicker := time.NewTicker(time.Second * 1)
	go p.calcMaxAndMinPacketNum(calcTicker)
	go p.checkComplete()
	prTicker := time.NewTicker(time.Duration(p.cfg.Interval) * time.Second)
	go p.pubAndRecOutput(prTicker)
	return nil
}

func (p *Publish) initData() error {
	p.lg.InfoC("开始组成员数据初始化...")
	err := p.database.C(config.CUser).Find(nil).All(&p.userData)
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
	for i, l := 0, len(p.userData); i < l; i++ {
		user := p.userData[i]
		clientID := user.UserID
		clientConn := NewHandleConnect(clientID, p)
		cli := client.New(&client.Options{
			ErrorHandler: clientConn.ErrorHandle,
		})
		connOptions := &client.ConnectOptions{
			Network:   p.cfg.Network,
			Address:   p.cfg.Address,
			ClientID:  []byte(clientID),
			KeepAlive: uint16(p.cfg.KeepAlive),
		}
		if p.cfg.UserName != "" && p.cfg.Password != "" {
			connOptions.UserName = []byte(p.cfg.UserName)
			connOptions.Password = []byte(p.cfg.Password)
		}
		if v := p.cfg.CleanSession; v {
			connOptions.CleanSession = v
		}
		err := cli.Connect(connOptions)
		if err != nil {
			return fmt.Errorf("Client %s connect error:%s", clientID, err.Error())
		}
		for j := 0; j < len(user.Groups); j++ {
			topic := "G/" + user.Groups[j]
			err = cli.Subscribe(&client.SubscribeOptions{
				SubReqs: []*client.SubReq{
					&client.SubReq{
						TopicFilter: []byte(topic),
						QoS:         p.cfg.Qos,
						Handler:     clientConn.Subscribe,
					},
				},
			})
			if err != nil {
				return fmt.Errorf("The client %s subscribe topic %s error:%s", clientID, topic, err.Error())
			}
		}
		p.clients[clientID] = cli
	}
	p.lg.InfoC("组成员建立MQTT数据连接完成.")
	return nil
}

func (p *Publish) checkComplete() {
	<-p.execComplete
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
		currentSecond := float64(ct.Sub(p.startTime)) / float64(time.Second)
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
		clientNum := len(p.clients) - p.disClients.Len()
		output := `
执行耗时                    %.2fs
执行次数                    %d
客户端数量                  %d
应发包量                    %d
实际发包量                  %d
每秒平均的发包量            %d
每秒最大的发包量            %d
应收包量                    %d
实际收包量                  %d
每秒平均的接包量            %d
每秒最大的接包量            %d`

		fmt.Printf(output,
			currentSecond, p.execNum, clientNum, p.publishTotalNum, p.publishNum, avgPNum, p.maxPublishNum, p.receiveTotalNum, p.receiveNum, avgRNum, p.maxReceiveNum)
		fmt.Printf("\n")
	}
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
	clientCount := len(p.clients)
	disCount := int(float32(clientCount) * (float32(p.cfg.DisconnectScale) / 100))
	for k, v := range p.clients {
		exist, _ := p.disClients.Contains(k)
		if exist {
			continue
		}
		v.Disconnect()
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
		p.userPublish(user)
		if v := p.cfg.UserInterval; v > 0 {
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}
}

func (p *Publish) userPublish(user config.User) {
	v, _ := p.disClients.Contains(user.UserID)
	if v {
		return
	}
	cli := p.clients[user.UserID]
	for i := 0; i < len(user.Groups); i++ {
		group := user.Groups[i]
		cTime := time.Now()
		sendPacket := config.SendPacket{
			SendID:   fmt.Sprintf("%d_%d", cTime.Unix(), atomic.AddInt64(&p.publishID, 1)),
			SendUser: user.UserID,
			ToGroup:  group,
			SendTime: cTime,
			Receives: make([]config.ReceivePacket, 0),
		}
		if p.cfg.IsStore {
			err := p.database.C(config.CPacket).Insert(sendPacket)
			if err != nil {
				p.lg.Errorf("Publish store error:%s", err.Error())
			}
		}
		bufData, err := json.Marshal(sendPacket)
		if err != nil {
			p.lg.Errorf("Publish json marshal error:%s", err.Error())
			continue
		}
		topic := "G/" + group
		err = cli.Publish(&client.PublishOptions{
			QoS:       p.cfg.Qos,
			TopicName: []byte(topic),
			Message:   bufData,
		})
		if err != nil {
			p.lg.Errorf("User %s publish topic %s error:%s", user.UserID, topic, err.Error())
			return
		}
		atomic.AddInt64(&p.publishNum, 1)
		if v := p.cfg.UserGroupInterval; v > 0 {
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}
}
