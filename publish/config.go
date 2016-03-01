package publish

type Config struct {
	ExecNum                int    // 执行次数
	Interval               int    // 发布间隔
	UserInterval           int    // 组成员发包间隔（单位毫秒）
	UserGroupInterval      int    // 组成员针对组的发包间隔（单位毫秒）
	AutoReconnect          bool   // 自动重连
	DisconnectScale        int    // 发送完成之后，断开客户端的比例
	IsStore                bool   // 是否执行持久化存储
	SendPacketStoreNum     int    // 发包一次性写入数据量
	SendPacketBucketNum    int    // 发包临时存储使用容器的数量
	ReceivePacketStoreNum  int    // 接包一次性写入数据量
	ReceivePacketBucketNum int    // 接包临时存储使用容器的数量
	MongoUrl               string // MongoDB连接
	Network                string
	Address                string
	Qos                    byte
	UserName               string
	Password               string
	CleanSession           bool
	KeepAlive              int
}
