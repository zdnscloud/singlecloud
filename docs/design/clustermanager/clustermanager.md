# 集群管理
## 概要
集群管理功能为singlecloud提供了对于kubernetes集群的基本管理功能，有效降低了管理员管理kubernetes集群的难度。
## 动机和目标
提供一个标准的rest api可以用来管理zcloud kubernetes集群，提供集群的创建、删除、增删节点等功能，该api要尽可能简洁，只要求一些安装zcloud kubernetes集群必须参数
## 架构
集群管理部分主要分为四个子模块：
* clusterManager：维护两个集群列表（ready、unready）、接收集群操作请求
* fsm：集群状态机，主要维护集群状态、集群event定义及相关callback
* zke：提供集群创建及更新接口
* validation：提供集群配置检查方法
因集群状态和集群相关的操作种类较多，故采用状态机的方式来管理一个集群的状态，集群状态转移图如下所示：
!["集群状态关系图"](fsm.jpg)
## 详细设计
### 集群状态分类
集群状态总体分为两大类：unready和ready，unready表示集群不可用，尚未初始化完成，unready状态下的集群的子资源接口不可用；ready表示集群已完成初始化，子资源接口可用
unready包括的状态有：Creating、Connecting、Updating、Unavailable、Canceling
ready包括的状态有：Unreachable、Running
### 权限验证
集群的创建、更新、删除、导入、获取集群kubeconfig接口只允许admin用户调用，普通用户只有对集群的查看权限
### 配置校验
* 创建集群
    * 集群option是否有空值：集群名称、ssh用户名、ssh私钥、singlecloud地址、集群域名后缀是否为空
    * 节点option校验：name、address和roles是否有空值；角色是否冲突（controlplane和worker互斥，etcd和edge不能单独存在，需依赖于controlplane或worker，即一个节点必须为controlplane或者worker节点）
    * 是否存在重复节点：两个节点的地址或名称相同
    * 节点数量校验：一个集群至少需要包含一个主节点、一个etcd节点和一个工作节点
    * 节点列表中是否包含了singlecloud server，若节点中包含singlecloud server，则不允许创建
* 更新集群
    * 节点option校验：name、address和roles是否有空值；角色是否冲突（controlplane和worker互斥，etcd和edge不能单独存在，需依赖于controlplane或worker，即一个节点必须为controlplane或者worker节点）
    * 是否存在重复节点：两个节点的地址或名称相同
    * 节点数量校验：一个集群至少需要包含一个主节点、一个etcd节点和一个工作节点
    * 节点列表是否包含老配置中的controlplane和etcd节点，contronplane和etcd节点只允许添加不允许删除
    * 节点列表中是否包含了singlecloud server，若节点中包含singlecloud server，则不允许创建
### 创建集群逻辑
* 检查是否存在同名集群
* 配置校验
* 创建cluster对象（状态为Creating），并将集群配置保存至db（此时db中的集群的Unavailable属性为true），将集群对象添加至unready列表
* 调用zke接口异步创建集群
    * 若创建成功，则向集群状态机发送创建成功事件，集群状态机执行callback更新db中该集群配置（添加集群证书配置）和db中集群的Unavailable属性为false同时更新内存中的集群状态为Running，并将集群move至ready列表中，通过singlecloud pubsub广播集群创建事件
    * 若创建失败，则向集群状态机发送创建失败事件，集群状态机更新内存中该集群状态为Unavailable
    * 若创建过程中收到集群取消请求，则zke安装线程会退出，并向集群状态机发送取消和取消成功事件，集群状态变为Unavailable
### 更新集群逻辑
* 检查是否存在该集群
* 配置校验
* 更新集群的节点配置（其他配置会忽略处理），将集群配置保存至db（此时db中的集群的Unavailable属性为true），更新集群状态（Updating），并将集群move至Unready列表，通过singlecloud pubsub广播集群删除事件
* 调用zke接口异步创建集群
    * 若更新成功，则向集群状态机发送更新成功事件，集群状态机执行callback更新db中该集群配置（更新集群证书配置）和db中集群的Unavailable属性为false同时更新内存中的集群状态为Running，并将集群move至ready列表中，通过singlecloud pubsub广播集群创建事件
    * 若更新失败，则向集群状态机发送更新失败事件，集群状态机更新内存中该集群状态为Unavailable
    * 若创建过程中收到集群取消请求，则zke线程会退出，并向集群状态机发送取消和取消成功事件，集群状态变为Unavailable
### 集群db加载
singlecloud启动时会读取db文件中的集群信息，并将其加载至内存中，一个集群的加载流程如下：
    * 从db中读取集群的信息（集群名称、Unavailable属性、配置及kubeconfig）
    * 根据读取的集群信息创建集群对象，并加入至unready列表中
        * 若集群的Unavailable属性为true，则将集群的状态设置为Unavailable
        * 若集群的Unavailable属性为false，则创建一个集群的initLoop线程执行该集群的初始化（创建kubeclient对象）
            * 若初始化成功，向集群状态机发送连接成功事件，集群状态机执行callback更新集群状态为Running，并将集群move至ready列表中，通过singlecloud pubsub广播集群创建事件
            * 若取消，则集群的initLoop线程会退出，并向集群状态机发送取消和取消成功事件，集群状态变为Unavailable
### 集群import逻辑
* 读取请求中的集群信息
* 检查是否存在同名集群
* 创建cluster对象（状态为connecting），并将集群配置保存至db（此时db中的集群的Unavailable属性为true），将集群对象添加至unready列表
* 创建一个该集群的initLoop线程执行该集群的初始化
    * 若初始化成功，向集群状态机发送连接成功事件，集群状态机执行callback更新集群状态为Running，并将集群move至ready列表中，通过singlecloud pubsub广播集群创建事件
    * 若取消，则集群的initLoop线程会退出，并向集群状态机发送取消和取消成功事件，集群状态变为Unavailable
### 获取集群kubeconfig
* 检查是否存在该集群
* 从db中读取该集群的kubeconfig配置，返回
## 未来工作
* 集群安装或更新进度实时反馈（websocket）
* 安装日志优化，更便于理解
* 安装前环境检测
* 集群配置校验细化
* 集群备份恢复功能
* 集群安装可选择kubernetes版本及集群kubernetes版本升级功能
