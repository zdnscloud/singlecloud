# 进程内的消息订阅机制
## 概要
提供进程内，模块之间的消息发布和订阅机制，降低模块之间的耦合

## 动机和目标
singlecloud是以资源为中心的管理系统，功能的扩展是通过定义新的资源来实现，而避免
修改已有资源的代码，系统中的资源之间可能独立，也可能存在包含，引用等关系.

有些重要资源比如workload，是系统中的重要资源，很多功能是围绕它展开，为了降低
重要资源本身的复杂性，我们通常定义新的资源来关联重要资源，也因此当重要资源创建
，变动或事删除以后，其他资源需要知道这些事件。

为了降低管理逻辑上相互关联的资源模块之间的耦合，通过资源事件发布和订阅的机制，实
现的这些模块之间数据同步。

## 架构
```text
                                +-------------+       +----------+   
                            +-->|event channel|-----> |subscriber|   
                            |   +-------------+       +----------+   
                            |                                        
+---------+       +-----+   |   +-------------+       +----------+   
|publisher ------>|topic|------>|event channel|------>|subscriber|   
+---------+       +-----+   |   +-------------+       +----------+   
                            |                                        
                            |   +-------------+       +----------+   
                            +-->|event channel|------>|subscriber|   
                                +-------------+       +----------+   
                                                                     
```

### pubsub模块
核心数据结构是维护一个topic到channel的映射，订阅和发布是通过这个映射来实现消息
的传递，主要接口
1. 订阅： `Sub(topics ...string) chan interface{}`, pubsub模块首先构造一个channel
然后将订阅者感兴趣的所有主题和新创建的channel建立映射
1. 发布消息: `Pub(msg interface{}, topics ...string)`, pubsub模块根据topic找到
对应的所有channel，并把消息推送到这些channel中
1. 取消订阅： `Unsub(ch chan interface{}, topics ...string)`, pubsub模块删除参数
中的channel和topic之间的映射关系

### eventbus模块
singlecloud在pubsub的基础上，对接口进行封装，建立以资源为中心的发布和订阅机制，
并且规范了消息的结构，主要支持3中资源消息：
- 资源创建
```go
type ResourceCreateEvent struct {
    Resource resource.Resource
}
```
- 资源修改
```go
type ResourceUpdateEvent struct {
    ResourceOld resource.Resource
    ResourceNew resource.Resource
}
```
- 资源删除
```go
type ResourceDeleteEvent struct {
    Resource resource.Resource
}
```
主要接口:
1. 订阅多个资源的消息：
`SubscribeResourceEvent(kinds ...resource.ResourceKind) chan interface{}`
 返回的值是一个只能读取的channel, channel中返回的三种event中的任意一种，注意是
结构体而不是结构体的指针。
1. 发布资源消息： 
- `PublishResourceCreateEvent(r resource.Resource)`
- `PublishResourceDeleteEvent(r resource.Resource)` 
- `PublishResourceUpdateEvent(resourceOld, resourceNew resource.Resource)`
1. 取消订阅资源消息
`UnsubscribeResourceEvent(chan interface{})`

## 未来工作
- 消息发送内部有消息缓存，所以当接受模块不能及时处理的时候，发送端不会阻塞，但是
当缓存填满，而订阅者还没有及时处理消息，就会阻塞发送者，目前没有加入告警机制
- 资源的其他类型的事件可能在将来的版本需要支持
