# zcloud的高可用
## 概要
保证zcloud的高可用，整个系统没有单点，当系统内任何节点出现硬件
故障，系统可用在分钟级别内恢复使用并且不丢失数据。

## 动机和目标
zcloud的后台服务进程singlecloud是有状态的，主要是导入的集群的信息，
用户信息和一些自定义资源的信息，singlecloud的高可用，最基本的是要
实现singlecloud的状态的高可用。

## 详细设计
web页面通过访问singlecloud来控制多个集群，singlecloud进程本身不存在
负载的问题，zcloud在部署的时候，使用主备的机制，浏览器一般情况下只能
访问主节点，singlecloud使用的是一个内嵌的实时备份的kv数据库kvzoo，
kvzoo在singlecloud中是一个单独的线程以减少部署的复杂度。
```text
                +---------------------+    +---------------------+
+------+        |singlecloud(master)  |    |singlecloud(slave)   |
|  IE  |--------> +----------------+  |    | +----------------+  |
+------+        | |kvzoo           |-------->|kvzoo           |  |
                | +----------------+  |    | +----------------+  |
                +---------------------+    +---------------------+
```

### 数据一致性
1. kvzoo在启动的时候，会检查主备节点的数据一致性，如果主备节点数据不一致
kvzoo会报错，singlecloud无法启动
1. singlecloud本身的数据格式的版本也会写入数据库中，每个singlecloud数据格式
版本的升级，都会提供数据迁移的工具，同时singcloud在启动过程中，也会检查当前
数据库的版本和singlecloud当前数据版本的一致性，如果不一致singlecloud同样无法
启动   

### singlecloud启动流程
singlecloudk只能以master角色或是slave角色启动，master角色启动时候可以指定
slave节点的地址以及db服务绑定的端口，slave角色启动的时候不需要指定master节点
同时slave节点不能再有其他slave，master节点启动了所有的组件和服务，slave节点
只启动db服务来实现数据同步。

### 数据备份和恢复流程
singlecloud的master节点启动之后需要制定slave节点地址以及kvzoo线程绑定的端口，
master节点每次数据写入都会实时同步到slave节点。 而启动slave节点时候不需要指定
master节点地址，同时也不能再指定其他的slave节点，当master节点宕机 或是硬盘故障，
管理员有两个选择:
1. 恢复master节点，并且从slave节点把数据拷贝到主节点
1. 当某个操作比较紧急，以master角色重新启动slave节点上的singlecloud浏览器修改访问
地址直接访问备份节点，当master节点恢复后， 在把slave节点的数据拷贝的master节点上。  

当采用第二种方法的时候，在主节点恢复之前整个系统处于状态的单点状态，所以
应该尽快恢复主节点。

kvzoo的所有数据都在一个文件中保存，所以同步数据的操作，就是通过单个文件，
但是在同步过程中系统不能进行任何写操作。
