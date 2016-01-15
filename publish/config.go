package publish

type Config struct {
	ExecNum           int    // 执行次数
	Interval          int    // 发布间隔
	UserInterval      int    // 组成员发包间隔（单位毫秒）
	UserGroupInterval int    // 组成员针对组的发包间隔（单位毫秒）
	DisconnectScale   int    // 发送完成之后，断开客户端的比例
	IsStore           bool   // 是否执行持久化存储
	MongoUrl          string // MongoDB连接
	Network           string
	Address           string
	Qos               byte
	UserName          string
	Password          string
	CleanSession      bool
	KeepAlive         int
}
