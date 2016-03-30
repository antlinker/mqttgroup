package publish

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/antlinker/go-mqtt/packet"

	"github.com/antlinker/go-mqtt/client"
	"github.com/antlinker/mqttgroup/config"
)

var hcrecvPacketCnt int64
var hcsendPacketCnt int64

func NewHandleConnect(clientID string, pub *Publish) *HandleConnect {
	handle := &HandleConnect{
		clientID: clientID,
		pub:      pub,
	}
	return handle
}

type HandleConnect struct {
	client.DefaultSubscribeListen
	client.DefaultPubListen
	client.DefaultDisConnListen
	clientID string
	pub      *Publish
}

func (hc *HandleConnect) OnLostConn(event *client.MqttEvent, err error) {
	hc.pub.lg.Debugf("OnLostConn:%v", err)
	if !hc.pub.cfg.AutoReconnect {
		hc.pub.clients.Remove(hc.clientID)
	}
}

func (hc *HandleConnect) OnRecvPublish(event *client.MqttRecvPubEvent, topic string, message []byte, qos client.QoS) {
	//hc.pub.lg.Debugf("OnRecvPublish:%s", topic)
	receiveTime := time.Now()
	atomic.AddInt64(&hc.pub.receiveNum, 1)
	if hc.pub.cfg.IsStore {
		var sendPacket config.SendPacket
		json.Unmarshal(message, &sendPacket)
		receivePacket := config.ReceivePacket{
			SendID:      sendPacket.SendID,
			ReceiveUser: hc.clientID,
			ReceiveTime: receiveTime,
		}
		hc.pub.receivePacketStore.Push(receivePacket)
		// err := hc.pub.database.C(config.CPacket).Update(bson.M{"sid": sendPacket.SendID}, bson.M{"$push": bson.M{"receives": receivePacket}})
		// if err != nil {
		// 	hc.pub.lg.Errorf("Handle subscribe store error:%s", err.Error())
		// }
	}
}

func (hc *HandleConnect) OnPubFinal(event *client.MqttPubEvent, mp *client.MqttPacket) {
	atomic.AddInt64(&hc.pub.publishNum, 1)
}

func (hc *HandleConnect) OnSubSuccess(event *client.MqttEvent, sub []client.SubFilter, result []client.QoS) {

	atomic.AddInt64(&hc.pub.subscribeNum, 1)
	hc.pub.clients.Set(hc.clientID, event.GetClient())
}
func (hc *HandleConnect) OnRecvPacket(event *client.MqttEvent, packet packet.MessagePacket, recvPacketCnt int64) {
	atomic.AddInt64(&hcrecvPacketCnt, 1)
	//hc.pub.lg.Debugf("OnRecvPacket:%d", rc)
}
func (hc *HandleConnect) OnSendPacket(event *client.MqttEvent, packet packet.MessagePacket, sendPacketCnt int64, err error) {
	atomic.AddInt64(&hcsendPacketCnt, 1)
	//hc.pub.lg.Debugf("OnSendPacket:%d", sc)
}
