package publish

type Config struct {
	ExecNum      int    // 执行次数
	Interval     int    // 发布间隔
	IsStore      bool   // 是否执行持久化存储
	MongoUrl     string // MongoDB连接
	Network      string
	Address      string
	Qos          byte
	UserName     string
	Password     string
	CleanSession bool
	KeepAlive    int
}
