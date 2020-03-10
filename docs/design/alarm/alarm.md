# 概要
将集群发生的异常事件和用户定义的资源预警，通过弹窗和邮件的方式及时告知用户


# 动机和目标
## 报警项
- 事件
	* kubernetes的Warning级别的event事件（暂时取消）
	* kubernetes集群里由cluster-agent发出reason为“core component abnormal”的event事件
	* singlecloud上集群/应用的操作导致的异步异常事件
- 资源
	* 管理员设置的集群资源预警指标

## 报警方式
- 铃铛
	
  铃铛显示当前所有未读告警数量（最大1000）
- 弹窗
	
  用户登录平台后发生的即时报警
- 邮件
	
  通过管理员设置的发件箱发送邮件到管理员指定的收件人


# 详细设计
## 报警资源
```
const (
        EventType  AlarmType = "Event"
        ZcloudType AlarmType = "Alarm"
)

type AlarmType string

type Alarm struct {
        resource.ResourceBase `json:",inline"`
        UID                   uint64           `json:"-"`
        Time                  resource.ISOTime `json:"time" rest:"description=readonly"`
        Cluster               string           `json:"cluster" rest:"description=readonly"`
        Type                  AlarmType        `json:"type" rest:"description=readonly"`
        Namespace             string           `json:"namespace" rest:"description=readonly"`
        Kind                  string           `json:"kind" rest:"description=readonly"`
        Name                  string           `json:"name" rest:"description=readonly"`
        Reason                string           `json:"reason" rest:"description=readonly"`
        Message               string           `json:"message" rest:"description=readonly"`
        Acknowledged          bool             `json:"acknowledged"`
}
```

## 指标配置
  
	管理员设置的集群级别的指标的保存为threshold
```
const (
        ThresholdConfigmapName         = "resource-threshold"
        ThresholdConfigmapNamespace    = "zcloud"
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
  1000

### 事件源
- cluster/application等资源是异步操作，当操作时发生异常，便会alarm.New()一个事件，发布到eventbus.AlarmEvent

  singlecloud运行后开始订阅eventbus.AlarmEvent，当有新消息时，则缓存到未读队列里，未读数加1，如果设置了邮箱便进行邮件报警
- 集群创建后开始监听kubernetes的event，当有create事件时，检查
  
  event.LastTimestamp 时间大于singlecloud的启动时间，且event.Reason为Cluster-agent创建event的reason为 "resource shortage"和"core component abnormal"
  
  如果都满足，则存到数据库中(最多存储1000条)，如果设置了邮箱便进行邮件报警

> 如果前后两条报警的Cluster、Namespace、Kind、Reason、Message、Name都一样，则忽略后一条报警

> 如果报警Kind为Cluster或者Node，则Namespace字段为空
```
func isRepeat(lastAlarm, newAlarm *types.Alarm) bool {
        return lastAlarm.Cluster == newAlarm.Cluster &&
                lastAlarm.Namespace == newAlarm.Namespace &&
                lastAlarm.Kind == newAlarm.Kind &&
                lastAlarm.Reason == newAlarm.Reason &&
                lastAlarm.Message == newAlarm.Message &&
                lastAlarm.Name == newAlarm.Name
}
```
```
var ClusterKinds = []string{"Node", "Cluster"}
if slice.SliceIndex(ClusterKinds, alarm.Kind) >= 0 {
                alarm.Namespace = ""
        }
```
## 数据返回
- 即时推送

	websocket连接建立后，推送当前未读数，并开始检查缓存的未读队列是否有新的消息，如果有消息或者未读数发生变化，便推送到前端
- 展示和标记

	报警展示返回数据库中的报警消息

  	用户可批量设置报警为已读，会将报警项在数据库中标记为已读

## 数据清理
- 监听集群的删除事件，集群删除时删除该集群的所有alarm
- 在内存和数据库增加alarm时检查当前缓存条数，超过上限时删除1个月之前的所有alarm，如果最早的一条alarm也是1个月之内的，便删除最早的1条
```
	oneMonthLater := time.Now().AddDate(0, -1, 0)
        delNum := 1
        for elem := ac.alarmList.Front().Next(); elem != nil; elem = elem.Next() {
                if oneMonthLater.Before(time.Time(elem.Value.(*types.Alarm).Time)) {
                        break
                }
                delNum += 1
        }
        firstID := ac.alarmList.Front().Value.(*types.Alarm).UID
        for i := 0; i < delNum; i++ {
                if err := deleteAlarmFromDB(ac.alarmsTable, uintToStr(uint64(i)+firstID)); err != nil {
                        return err
                }
                elem := ac.alarmList.Front()
                ac.alarmList.Remove(elem)
        }
```

# TODO
- 根据权限定向报警
- 增加报警级别
- 名称空间的报警发送到改空间的所有者的邮箱
- Kubernetes的event目前没有很好的筛选条件，无法将重要信息推送给用户
