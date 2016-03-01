# MQTT针对群组消息的压力测试

## 获取

``` bash
$ go get github.com/antlinker/mqttgroup
```

## 使用

``` bash
$ cd github.com/antlinker/mqttgroup
$ go build -o mqttgroup
```

## Help

``` bash
$ ./mqttgroup help
```

```
NAME:
   mqttgroup - MQTT群组聊天测试

USAGE:
   ./mqttgroup [global options] command [command options] [arguments...]

VERSION:
   0.1.0

AUTHOR(S):
   Lyric

COMMANDS:
   generate, gen        生成群组并且分配群成员
   publish, pub         发布消息
   clear, c             清除数据
   collect              汇总接收的包数据
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h           show help
   --version, -v        print the version
```

### publish

```
NAME:
   ./mqttgroup publish - 发布消息

USAGE:
   ./mqttgroup publish [command options] [arguments...]

OPTIONS:
   --ExecNum, --en "10"                         执行次数
   --Interval, -i "5"                           发布间隔(单位:秒)
   --UserInterval, --ui "1"                     组成员发包间隔（单位:毫秒）
   --UserGroupInterval, --ugi "0"               组成员针对组的发包间隔（单位:毫秒）
   --AutoReconnect, --ar "1"                    客户端断开连接后执行自动重连(默认为1，0表示不重连)
   --DisconnectScale, --ds "0"                  发送完成之后，需要断开客户端的比例
   --IsStore, -s                                是否执行持久化存储
   --SendPacketStoreNum, --spsn "200"           发包一次性写入数据量
   --SendPacketBucketNum, --spbn "100"          发包临时存储使用容器的数量
   --ReceivePacketStoreNum, --rpsn "1000"       接包一次性写入数据量
   --ReceivePacketBucketNum, --rpbn "100"       接包临时存储使用容器的数量
   --Network, --net "tcp"                       MQTT Network
   --Address, --addr "127.0.0.1:1883"           MQTT Address
   --UserName, --name                           MQTT UserName
   --Password, --pwd                            MQTT Password
   --QOS, --qos "1"                             MQTT QOS
   --KeepAlive, --alive "30"                    MQTT KeepAlive
   --CleanSession, --cs                         MQTT CleanSession
   --mongo, --mgo "mongodb://127.0.0.1:27017"   MongoDB连接url
```

