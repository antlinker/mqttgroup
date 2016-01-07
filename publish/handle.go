package publish

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/ant-testing/mqttgroup/config"
)

func NewHandleSubscribe(clientID string, pub *Publish) *HandleSubscribe {
	return &HandleSubscribe{
		clientID: clientID,
		pub:      pub,
	}
}

type HandleSubscribe struct {
	clientID string
	pub      *Publish
}

func (hs *HandleSubscribe) Handler(topicName, message []byte) {
	atomic.AddInt64(&hs.pub.receiveNum, 1)
	if hs.pub.cfg.IsStore {
		var sendPacket config.SendPacket
		json.Unmarshal(message, &sendPacket)
		receivePacket := config.ReceivePacket{
			ReceiveUser: hs.clientID,
			ReceiveTime: time.Now(),
		}
		err := hs.pub.database.C(config.CPacket).Update(bson.M{"sid": sendPacket.SendID}, bson.M{"$push": receivePacket})
		if err != nil {
			hs.pub.lg.Errorf("Handle subscribe store error:%s", err.Error())
		}
	}
}
