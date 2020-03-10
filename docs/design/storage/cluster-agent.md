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
获取集群pv信息

### 接口
- get
- list

### 原理
- 通过监听api上的pv、pvc、pod资源，得到pv与pvc对应关系，pvc与pod的对应关系
- 通过向所有node-agent发出GetMountpointsSize请求(lsblk --bytes -J -o NAME,TYPE,SIZE,PKNAME,FSTYPE,MOUNTPOINT  )，获取主机上所有挂载点信息（挂载设备、设备大小、设备使用情况、挂载点路径）
- 再根据以上两个信息，汇总得到每种storagelcass其各自的pv的信息（空间使用情况、pod、node）

# 缓存

## 原因
  上面两个资源由于都需要到所有节点执行命令，延时较长
## 原理
github.com/zdnscloud/cement/cache
```
cache:        cementcache.New(1, hashBlockdevices, false),
var key = cementcache.HashString("1")
func hashBlockdevices(s cementcache.Value) cementcache.Key {
        return key
}
```
请求时优先GetBuf()，如果缓存内没有数据，再调用SetBuf()
```
func (m *blockDeviceMgr) SetBuf() BlockDevices {
        bs := m.getBlockdevicesFronNodeAgent()
        if len(bs) == 0 {
                log.Warnf("Has no blockdevices to cache")
                return bs
        }
        m.cache.Add(&bs, time.Duration(m.timeout)*time.Second)
        return bs
}

func (m *blockDeviceMgr) GetBuf() BlockDevices {
        log.Infof("Get blockdevices from cache")
        var bs BlockDevices
        res, has := m.cache.Get(key)
        if !has {
                log.Warnf("Cache not found blockdevice")
                return bs
        }
        bs = *res.(*BlockDevices)
        return bs
}
```

>   缓存时长默认为60秒，也可通过设置环境CACHE_TIME来更改
