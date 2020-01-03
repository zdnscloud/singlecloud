# 概要
将集群发生的异常事件，通过弹窗和邮件的方式及时告知用户


# 动机和目标
## 报警项
- 事件
	* kubernetes的Warning级别的event事件
	* singlecloud上集群/应用的操作导致的异步异常事件
- 资源
	* 管理员设置的集群级别的资源指标和用户设置的名称空间的资源告警指标

## 报警方式
- 铃铛
	
  铃铛显示当前所有未读告警数量（最大100）
- 弹窗
	
  用户登录平台后发生的即时报警
- 邮件
	
  如果用户设置了资源告警指标并配置了邮件发送者和接受者，管理员设置的邮箱会收到所有报警邮件；普通用户设置的邮箱只会受到该名称空间下发生的报警邮件
> 集群/节点报警只会发送管理员邮箱


# 详细设计

## 指标配置
	用户设置的指标会保存在kubernetes 的zcloud名称空间下，以configmap方式保存
  
	管理员设置的集群级别的指标的保存为resource-threshold，普通用户的针对名称空间的指标保存为resource-threshold-XXXX（namespace名字）
```
const (
        ClusterThresholdConfigmapName         = "resource-threshold"
        NamespaceThresholdConfigmapNamePrefix = "resource-threshold-"
        ThresholdConfigmapNamespace           = "zcloud"
)
```  
	Cluster-agent运行后会监听上面的configmap，在得到具体指标后按照60s的固定频率进行检查，如果有超过指标的检查项，则创建一个kubernetes的Warning级别event

```
const (
        CheckInterval           = 60
)
```
## 缓存

### 缓存长度
  100

### 事件源
- cluster/application等资源是异步操作，当操作时发生异常，便会alarm.New()一个事件，发布到eventbus.AlarmEvent

  singlecloud运行后开始订阅eventbus.AlarmEvent，当有新消息时，则缓存到未读队列里，未读数加1，如果设置了邮箱便进行邮件报警
- 集群创建后开始监听kubernetes的event，当有create事件时，检查

  event.Type为Warning
  
  event.InvolvedObject.Kind是以下列出的：
  * Cluster
  * Node
  * Namespace
  * Pod
  * StatefulSet
  * Deployment
  * DaemonSet
  * StorageClass
  * PersistentVolume
  * PersistentVolumeClaim
  
  如果都满足，则缓存到未读队列里，未读数加1，如果设置了邮箱便进行邮件报警
```
var EventLevelFilter = []string{corev1.EventTypeWarning}
var EventKindFilter = []string{
        "Cluster",
        "Node",
        "Namespace",
        "Pod",
        "StatefulSet",
        "Deployment",
        "DaemonSet",
        "StorageClass",
        "PersistentVolume",
        "PersistentVolumeClaim",
}
```
## 数据返回
- 即时推送

	websocket连接建立后，推送当前未读数，并开始检查缓存的未读队列是否有新的消息，如果有消息或者未读数发生变化，便推送到前端
- 展示和标记

	报警展示返回100条报警，优先返回未读队列，不足的再从已读队列取。
```
        for e := m.cache.alarmList.Back(); e != nil; e = e.Prev() {
                alarms = append(alarms, e.Value.(*types.Alarm))
        }
        for e := m.cache.ackList.Back(); len(alarms) < int(m.cache.maxSize) && e != nil; e = e.Prev() {
                alarms = append(alarms, e.Value.(*types.Alarm))
        }
```
  用户可批量设置报警为已读，会将报警项从未读队列移动到已读队列，未读数减1
```
                        m.cache.alarmList.Remove(e)
                        m.cache.SetUnAck(-1)
                        newAlarm.Acknowledged = true
                        m.cache.ackListAdd(newAlarm)
```

# TODO
- 根据权限定向报警
- 增加报警级别
