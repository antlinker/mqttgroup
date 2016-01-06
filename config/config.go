package config

import (
	"time"
)

const (
	Database = "group-testing"
	CGroup   = "group"
	CUser    = "user"
	CPacket  = "packet"
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
	SendID   string          `bson:"sid"`
	SendUser string          `bson:"uid"`
	ToGroup  string          `bson:"gid"`
	SendTime time.Time       `bson:"time"`
	Receives []ReceivePacket `bson:"receives"`
}

type ReceivePacket struct {
	ReceiveUser string    `bson:"uid"`
	ReceiveTime time.Time `bson:"time"`
}
