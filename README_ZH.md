# server

该示例是基于RuleGo中的server示例改编而成，并实现RuleGo的功能且提供示例。

前端在线调试界面：[example.rulego.cc](https://example.rulego.cc/) 。

该工程提供以下功能：

* 执行规则链并得到执行结果API
* 往规则链上报数据API，不关注执行结果。
* 创建规则链API。
* 更新规则链API。
* 获取节点调试日志API。
* 执行规则链并得到执行结果API。
* 实时推送执行日志。
* 保存执行快照。
* 组件列表API。
* 订阅MQTT数据，并根据根规则链定义交给规则引擎处理。
* 规则链数据持久化。
* 实现了自定义组件xlsx2json，并提供了运行示例。
* 共享资源实现并提供运行示例。

## HTTP API

* 获取所有组件列表
    - GET /api/v1/components

* 执行规则链并得到执行结果API
    - POST /api/v1/rule/:chainId/execute/:msgType
    - chainId：处理数据的规则链ID
    - msgType：消息类型
    - body：消息体
  
* 往规则链上报数据API，不关注执行结果
  - POST /api/v1/rule/:chainId/notify/:msgType
  - chainId：处理数据的规则链ID
  - msgType：消息类型
  - body：消息体
  
* 查询规则链
    - GET /api/v1/rule/{chainId}/{nodeId}
    - chainId：规则链ID
    - nodeId:空则查询规则链定义，否则查询规则链指定节点ID节点定义

* 保存或更新规则链
    - POST /api/v1/rule/{chainId}/{nodeId}
    - chainId：规则链ID
    - nodeId：空则更新规则链定义，否则更新规则链指定节点ID节点定义
    - body：更新内容
  
* 保存规则链Configuration
    - POST /api/v1/rule/:chainId/saveConfig/:varType
    - chainId：规则链ID
    - varType: vars/secrets 变量/秘钥
    - body：配置内容

* 获取节点调试日志API
    - Get /api/v1/event/debug?&chainId={chainId}&nodeId={nodeId}
    - chainId：规则链ID
    - nodeId：节点ID

  当节点debugMode打开后，会记录调试日志。目前该接口日志存放在内存，每个节点保存最新的40条，如果需要获取历史数据，请实现接口存储到数据库。

## server编译

为了节省编译后文件大小，默认不引入扩展组件[rulego-components](https://github.com/rulego/rulego-components) ，默认编译：

```shell
cd cmd/server
go build .
```

如果需要引入扩展组件[rulego-components](https://github.com/rulego/rulego-components) ，使用`with_extend`tag进行编译：

```shell
cd cmd/server
go build -tags with_extend .
```
其他扩展组件库tags：
- 注册扩展组件[rulego-components](https://github.com/rulego/rulego-components) ，使用`with_extend`tag进行编译：
- 注册AI扩展组件[rulego-components-ai](https://github.com/rulego/rulego-components-ai) ，使用`with_ai`tag进行编译
- 注册CI/CD扩展组件[rulego-components-ci](https://github.com/rulego/rulego-components-ci) ，使用`with_ci`tag进行编译
- 注册IoT扩展组件[rulego-components-iot](https://github.com/rulego/rulego-components-iot) ，使用`with_iot`tag进行编译

如果需要同时引入多个扩展组件库，可以使用`go build -tags "with_extend,with_ai,with_ci,with_iot" .` tag进行编译。

## server启动

```shell
./server -c="./config.conf"
```

或者后台启动

```shell
nohup ./server -c="./config.conf" >> console.log &
```
