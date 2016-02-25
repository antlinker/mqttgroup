package collect

// Config 配置信息
type Config struct {
	StoreNum int    // 批量写入MongoDB的数据条数
	MongoUrl string // MongoDB连接
}
