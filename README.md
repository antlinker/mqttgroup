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
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h           show help
   --version, -v        print the version
```

