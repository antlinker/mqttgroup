package generate

type Config struct {
	GroupNum            int    // 组数量
	GroupClientLimitNum int    // 组成员数量限制
	ClientNum           int    // 组成员数量
	ClientGroupLimitNum int    // 组成员所拥有的组数量限制
	MongoUrl            string // MongoDB连接
}
