package config

import (
	"time"
)

const (
	Database       = "mqttgroup"
	CGroup         = "groups"
	CUser          = "user"
	CPacket        = "sendpacket"
	CReceivePacket = "receivepacket"
	CCollect       = "collectpacket"
)

type Group struct {
	GroupID string `bson:"gid"`
	Weight  int    `bson:"-"`
}

type User struct {
	UserID string   `bson:"uid"`
	Groups []string `bson:"groups"`
}

type SendPacket struct {
	SendID   string    `bson:"sid"`
	SendUser string    `bson:"uid"`
	ToGroup  string    `bson:"gid"`
	SendTime time.Time `bson:"time"`
}

type ReceivePacket struct {
	SendID      string    `bson:"sid"`
	ReceiveUser string    `bson:"uid"`
	ReceiveTime time.Time `bson:"time"`
}

// CollectPacket 包汇总
type CollectPacket struct {
	PacketID      string  `bson:"packetid"`
	PreReceiveNum int64   `bson:"prerecvnum"`
	ReceiveNum    int64   `bson:"recvnum"`
	MaxConsume    float64 `bson:"max"`
	MinConsume    float64 `bson:"min"`
	AvgConsume    float64 `bson:"avg"`
}
