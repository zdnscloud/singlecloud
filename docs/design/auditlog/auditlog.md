# Auditlog
## 概要
对singlecloud的相关操作进行记录、存储，满足产品日志审计要求
## 动机和目标
一款商业化管理软件需要满足基本的日志审计要求，即需要记录在该平台上执行过的操作，在需要时可以查看、审计操作记录
需记录除查看外的所有操作，日志信息需包括：用户、请求源地址、操作类型（创建、更新、删除、登录等）、操作的资源类型、操作的资源PATH、请求参数、操作时间
# 详细设计
## Auditlog资源定义
```go
type AuditLog struct {
	resource.ResourceBase `json:",inline"`
	UID                   uint64 `json:"uid"`
	User                  string `json:"user"`
	SourceAddress         string `json:"sourceAddress"`
	Operation             string `json:"operation"`
	ResourceKind          string `json:"resourceKind"`
	ResourcePath          string `json:"resourcePath"`
	Detail                string `json:"detail"`
}
```
其中`uid`参数为audit模块做日志轮滚使用
## 日志记录
Audit模块通过实现gorest handlerFunc接口，作为gorest的一个中间件对所有经过gorest的请求进行日志记录
* handlerFunc实现
```go
func (a *AuditLogger) AuditHandler() gorest.HandlerFunc {
	return func(ctx *resource.Context) *resterr.APIError {
		log := &types.AuditLog{
			User:          getCurrentUser(ctx),
			SourceAddress: ctx.Request.Host,
			ResourceKind:  resource.DefaultKindName(ctx.Resource),
			ResourcePath:  ctx.Request.URL.Path,
		}
        ...
		return nil
	}
}
```
* 在gorest中使用audit
```go
server.Use(auditLogger.AuditHandler())
```
> 若gorest参数验证失败，则请求不会经过audit模块，不会记录；业务代码中自定义的校验逻辑返回的错误audit无法感知，所以audit不会记录操作结果
## 持久化
audit对象实例包含一个audit storage接口：
```go
type AuditLogger struct {
	Storage storage.StorageDriver
}
```
StorageDriver接口实现了Add和List接口：
```go
type StorageDriver interface {
	Add(a *types.AuditLog) error
	List() (types.AuditLogs, error)
}
> StorageDriver在Add时会加锁，List操作无锁
```
StorageDriver在每次Add时检查日志条数是否已达到最大条数限制，若达到，则先删除最老的一条日志，再写入新的日志，已有日志条数通过StorageDriver中的firstID和currentID进行判断
```go
func (d *DefaultDriver) Add(a *types.AuditLog) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.currentID-d.firstID > uint64(d.maxRecordCount-2) {
		if err := deleteFromDB(d.table, uintToStr(d.firstID)); err != nil {
			return err
		}
		atomic.AddUint64(&d.firstID, 1)
	}
	a.UID = d.currentID + 1
	a.SetID(uintToStr(d.currentID + 1))
	if err := addToDB(d.table, a); err != nil {
		return err
	}
	atomic.AddUint64(&d.currentID, 1)
	return nil
}
```
## Todo
* 支持推送至第三方日志服务器
* 支持分页，增加存储的最大条目
* 支持记录操作结果
