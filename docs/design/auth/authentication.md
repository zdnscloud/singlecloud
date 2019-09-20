# 用户身份验证
## 概要
回答当前用户是谁的问题，用户通过用户名来标识

## 动机和目标
对于web用户, 支持cas和用户和密码认证两种方式
对于api用户, 只支持用户明和密码认证
通过认证模块在请求的上下文中设置用户名，使得后面的业务逻辑
能过获取用户身份信息

## 详细设计
支持Json Web Tokens(JWT)和Central Authentication Service(CAS)两种认证协议
web请求和api请求使用分离的验证流程

## JWT验证
系统保存用户名以及使用SHA1哈希之后的密码，验证请求通过在body段携带用户吗和哈希
之后的密码，当用户名和密码匹配之后，JWT模块使用HS256(HMAC/SHA256)对用户名和失效
时间通过哈希之后用配置的密钥生成签名数据，经过base64编码生成token. 验证过程通过
密钥解密签名数据，然后通过hash对比来验证用户名和失效时候的有效性。 

### web登录验证
用户在登录页面输入用户名和密码，前端js会用POST方法异步调用调用url(/web/login)
jwt模块验证用户名和密码正确后，生成对应token，并将token放入session中

### API登录验证
API用户访问url（apis/zcloud.cn/v1/users/:user_name?action=login)获取token，请求
的body段携带用户密码，验证成功后，返回token，之后的请求需要携带token在header中

## CAS验证
在启动singlecloud的时候通过cas使能cas验证，参数是cas服务的地址

### web认证
系统检测到未验证的请求回自动跳转到cas服务的地址，并且把/web/casredirect作为cas
服务的回掉url，当在cas页面认证成功后会进入回掉页面，在页面中会把cas返回的ticket
写入用户的session中

## 认证检查
所有用户请求，除访问以下页面外，都会尝试认证用户
 - 静态资源: /asserts
 - websocket: /apis/ws.zcloud.cn
 - web认证页面: /web
 - cas跳跳
如果不能获取用户信息，对于除访问系统api和正要做web登陆的请求实现自动跳转
 - 对于系统使能cas认证方式的，跳转到配置的cas认证页面
 - 默认跳转到/login页面
通过验证的用户，将当前用户的用户名写入请求的上下文

## session管理
由于singlecloud是一个企业的管理系统，注册使用的用户可控，所以session数据保存在内存中
jwt的cookie名称为_jwt_session， cas模块的cookie名称为_cas_session

## 架构设计
验证模块在rest资源处理逻辑之前，作为一个gin的middleware来处理请求
api用户的登录是通过user资源的login action来实现
web和api验证共享同一个jwt验证模块

## 未来工作
* jwt的密钥目前是硬编码，出于安全的考虑，需要支持通过配置指定
