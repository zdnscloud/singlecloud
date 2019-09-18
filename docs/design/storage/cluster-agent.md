# 资源
## blockdevice  

### 用途
获取主机磁盘信息
### 接口  
- list

### 原理
通过向所有node-agent发出GetDisksInfo请求(df)，获取主机上的块设备信息（磁盘大小、是否分区、是否有文件系统、是否有挂载点）

## storage 
### 用途
获取集群两种storagetype的pv信息

### 接口
- get
- list

### 原理
- 通过监听api上的pv、pvc、pod资源，得到pv与pvc对应关系，pvc与pod的对应关系
>  创建lvm和cephfs两个monitor，分别监听属于各自storagetype的pv和pvc

>  判断pv和pvc所属的storagetype需要用到lvm和cephfs存储其固定的provisioner名称

- 通过向所有node-agent发出GetMountpointsSize请求(lsblk --bytes -J -o NAME,TYPE,SIZE,PKNAME,FSTYPE,MOUNTPOINT  )，获取主机上所有挂载点信息（挂载设备、设备大小、设备使用情况、挂载点路径）
- 再根据以上两个信息，汇总得到两种storagetype其各自的pv的信息（空间使用情况、pod、node）
