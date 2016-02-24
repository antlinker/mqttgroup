package publish

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/antlinker/mqttgroup/config"
)

func NewHandleConnect(clientID string, pub *Publish) *HandleConnect {
	handle := &HandleConnect{
		clientID:    clientID,
		pub:         pub,
		receiveChan: make(chan []byte, 10),
	}
	if pub.cfg.IsStore {
		go handle.HandleReceive()
	}
	return handle
}

type HandleConnect struct {
	clientID    string
	receiveChan chan []byte
	pub         *Publish
}

func (hc *HandleConnect) ErrorHandle(err error) {
	hc.pub.lg.Errorf("客户端%s发生异常:%s,已断开连接!", hc.clientID, err.Error())
	if !hc.pub.cfg.AutoReconnect {
		hc.pub.clients.Remove(hc.clientID)
	}
}

func (hc *HandleConnect) Subscribe(topicName, message []byte) {
	atomic.AddInt64(&hc.pub.receiveNum, 1)
	if hc.pub.cfg.IsStore {
		hc.receiveChan <- message
	}
}

func (hc *HandleConnect) HandleReceive() {
	for msg := range hc.receiveChan {
		var sendPacket config.SendPacket
		json.Unmarshal(msg, &sendPacket)
		receivePacket := config.ReceivePacket{
			ReceiveUser: hc.clientID,
			ReceiveTime: time.Now(),
		}
		err := hc.pub.database.C(config.CPacket).Update(bson.M{"sid": sendPacket.SendID}, bson.M{"$push": bson.M{"receives": receivePacket}})
		if err != nil {
			hc.pub.lg.Errorf("Handle subscribe store error:%s", err.Error())
		}
	}
}

func (hc *HandleConnect) Close() {
	if hc.receiveChan != nil {
		close(hc.receiveChan)
	}
}
