# node隔离和驱逐
## 概要
在需要对node进行排查或下线维护时，提供node隔离（停止调度）和node驱逐（驱逐该node上的pod）的功能。
## 动机和目标
在大规模k8s集群环境中，node故障（硬件、操作系统或系统组件故障）是最常见的故障之一，所以需要提供在该场景下，停止向该node调度pod和驱逐该node上运行的pod（以便于对node进行维护）的功能。
## 详细设计
利用k8s scheduler原理，分别对node添加不同的taint实现对node停止调度以及驱逐的效果。
cordon、drain、uncordon均为node的`action`操作。
* cordon
    * 含义：停止向node调度pod
    * 实现方式：将node的unschedulable属性置为true（k8s会自行为node添加NoSchedule的taint）,此时singlecloud get node的状态为`cordoned`
```yaml
spec:
  podCIDR: 10.42.1.0/24
  taints:
  - effect: NoSchedule
    key: node.kubernetes.io/unschedulable
    timeAdded: "2019-10-28T07:47:49Z"
  unschedulable: true
```
* drain
    * 含义：停止向node调度pod同时驱逐node上已运行的pod
    * 实现方式：将node的unschedulable属性置为true同时为node添加effect为NoExecute的taint，此时singlecloud get node的状态为`drained`
```yaml
spec:
  podCIDR: 10.42.1.0/24
  taints:
  - effect: NoSchedule
    key: node.kubernetes.io/unschedulable
    timeAdded: "2019-10-28T07:47:49Z"
  - effect: NoExecute
    key: node.zcloud.cn/unexcutable
    timeAdded: "2019-10-28T07:47:50Z"
  unschedulable: true
```
* uncordon
    * 含义：恢复node调度并取消node drained状态（若node没有执行驱逐操作则只恢复调度）
    * 实现方式：将node的unschedulable属性置为true同时移除node上effect为NoExecute的taint（若存在）
* 为什么drain没有对应的undrain，而是和cordon共用uncordon操作恢复？
    1. 无场景需要：node保留驱逐状态的情况即使移除NoSchedule的taint也不会有pod调度到该node上，除非为pod添加相应的toleration
    2. 针对node的cordon、drain、uncordon操作由kubectl命令而来，沿用k8s生态已有的命名和操作习惯，避免新增学习成本。
### 与kubectl命令差异
cordon以及uncordon完全相同，差异主要在于drain，kubectl的drain提供了更多的参数选择:
1. 可选是否删除pod的临时数据（即pod使用的emptyDir），默认保留
2. 强制删除，默认kubectl drain不会删除node上单独的pod（没有被ReplicationController管理的孤儿pod），默认为false
3. 是否忽略daemonsets，daemonsets的pod无法被驱逐，默认不忽略，kubectl会有报错提示信息
4. 可通过pod-selector指定驱逐特定的pod
5. 超时时间，到达设定的超时时间若kubectl还未完成对所有pod的驱逐，则结束本次drain，默认一直等待
* kubectl drain对pod的驱逐实现是通过client先get node上所有的pod，然后根据条件筛选出需要驱逐的pod，再通过client逐一对所需pod进行驱逐，优点是可以实现node驱逐的同步，即kubectl命令执行完后，node上的pod已经驱逐完成，可以直接将此node下线而不会影响业务。
## Todo
* node drain的完成进度反馈