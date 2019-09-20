
singlecloud api
======

# Terminology
* APIVersion: api版本，包含Group和Version两个字段，代表api的使用群组和版本信息

* Resource: 由api操作的基础资源对象，每个基础资源对象包含基本属性和每个资源特有属性，基础属性包含id，资源类型type，资源链接links，创建时间creationTimestamp
  * Collection为相同类型Resource集合，Collection名字根据Resource名字自动生成，规则与英文单词复数一致
  	 * 如果Resource名字以字母s、ch、x结尾，加es
  	 * 如果Resource名字以字母y结尾，且y前面那个字母不是［aeiou］，将y变成i加es
  	 * 其他直接加s
  * 资源之间父子关系
    * 顶级资源: cluster user user_quota registry
    * cluster子资源：blockdevice  monitor  namespace  node  nodenetwork persistentvolume podnetwork  servicenetwork  storageclass storagecluster
    * namespace子资源:  application  chart  configmap  cronjob deployment daemonset ingress innerservice  job limitrange  outservice  persistentvolumeclaim  resourcequota  secret  service  statefulset udpingress
    * deployment/daemonset/satefulset子资源: pod
	
* Collection: 代表一组相同类型的Resource，当用户获取一种类型的所有Resource时，就会返回Collection，其中Collection的type为collection，字段resourceType为Data资源数组的资源类型，data为Resource资源对象数组

* Action: 对资源的一种操作，使用POST方式，但是又不是像POST添加资源的操作，例如Login这种特殊的操作，其中字段name为Action名字，input为Action的参数，output为Action返回的结果

* JSON Web Tokens（JWT）: 是一个开放标准（rfc 7519） 定义的一种紧凑的、自包含的方式在json对象之间安全地传输信息。因为它是数字签名的，所以它是可以验证并且可信的。jwt可以使用密钥（使用hmac算法）或使用rsa或ecdsa的公钥/私钥对进行签名。用户通过使用user的login接口，传递正确的用户名和密码哈希值获取token，密码的哈希算法是sha1，然后使用16进制编码，后续除login以外的请求都要带上token，服务端会根据token验证有效性。携带token的方式是设置HTTP Header的Authorization值，Authorization的值为 Bearer+空格+token，admin的默认密码为zcloud，编码后为：0192309fba8c6f0929b5b00867ebccac9a39e34e, curl案例如下：

		curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"


# URL
  * 资源URL由groupPrefix、APIVersion和资源父子关系组成, 目前支持的groupPrefix只有/apis， APIVersion包含group和version两个字段，group目前只支持zcloud.cn，version为v1
  * 如果资源是顶级资源，没有父资源，如cluster，自动生成URL为 /apis/zcloud.cn/v1/clusters
  * 如果资源只有一个父资源，如namespace父资源为cluster，那么自动生成URL为 /apis/zcloud.cn/v1/clusters/cluster_id/namespaces
  * 如果资源有多个父资源，那么会自动生成多个URL， 如pod父资源有deployment、daemonset、statefulset，自动生成的URL为
  
  /apis/zcloud.cn/v1/clusters/cluster_id/namespaces/namespace_id/deployments/deployment_id/pods 
  /apis/zcloud.cn/v1/clusters/cluster_id/namespaces/namespace_id/daemonsets/daemonset_id/pods 
  /apis/zcloud.cn/v1/clusters/cluster_id/namespaces/namespace_id/statefulsets/statefulset_id/pods
    
# Operations

* Create Operation: 创建一个Resource
  * Request: 
    * HTTP Method: POST http://host/apis/zcloud.cn/v1/{collection_name}
    * Http Header 中使用Authorization Bearer方式携带用户密码 
    * body parameters: 所有参数定义fields中，如果有嵌套结构，则子字段放在subResources中
  * Response: 
    * status code: 201 Created 或者其他错误code
    * body: 返回一个Resource 
* Delete Operation: 删除一个已存在的Resource
   * Request: 
  	* HTTP Method: DELETE http://host/apis/zcloud.cn/v1/{collection_name}/{resource_id} 
  	* Http Header 中使用Authorization Bearer方式携带用户密码 
  * Response: 
    * status code: 204 No Content 或者其他错误code  
* Update Operation: 更新一个已存在的Resource属性，目前只支持全量更新
  * Request: 
  	* HTTP Method: PUT http://host/apis/zcloud.cn/v1/{collection_name}/{resource_id} 
  	* Http Header 中使用Authorization Bearer方式携带用户密码 
    * body parameters: 所有参数定义fields中，如果有嵌套结构，则子字段放在subResources中
  * Response: 
    * status code: 200 OK 或者其他错误code
    * body: 返回更新后的Resource 
* List Operation: 返回一种类型Resource的Collection
  * Request: 
  	* HTTP Method: GET http://host/apis/zcloud.cn/v1/{collection_name}
  	* Http Header 中使用Authorization Bearer方式携带用户密码 
  * Response: 
    * status code: 200 OK 或者其他错误code
    * body: 返回一个collection
* Get Operation: 获取一个Resource
  * Request: 
  	* HTTP Method: GET http://host/apis/zcloud.cn/v1/{collection_name}/{resource_id} 
  	* Http Header 中使用Authorization Bearer方式携带用户密码 
  * Response: 
    * status code: 200 OK 或者其他错误code
    * body: 返回一个Resource
* Action Operation: 执行一种action操作
  * Request:
    * HTTP Method: POST http://host/apis/zcloud.cn/v1/{collection_name}/{resource_id}?action={action_name}
    * Http Header 中使用Authorization Bearer方式携带用户密码 
    * body parameters: 一个object，定义需要的字段
  * Response: 
    * status code: 200 OK 或者其他错误code
    * body: 一个string

# Status Code
api的应答会包含Http status code，请求成功会返回2xx，请求失败会返回4xx或5xx

* 200 OK, 更新成功，Action操作成功，获取资源没有报错都返回200
* 201 Created，创建资源成功返回201
* 204 NoContent, 删除成功返回204
* 401 Unauthorized, 认证失败返回401
* 403 Forbidden，鉴权失败返回403
* 404 NotFound，资源未找到返回404
* 405 MethodNotAllow，资源不支持的操作返回405
* 409 Conflict，资源操作发生冲突返回409
* 422 UnprocessableEntity，其他错误返回422
* 500 InternalServerError，内部错误返回500
* 503 ClusterUnavailable， 集群不可用时返回503

# Links
  * 操作资源时response会有links字段返回，方便client快捷使用，如statefulset的id为sts123的资源links如下

		{
			"links": {
        		"collection": "http://host/apis/zcloud.cn/v1/clusters/beijing/namespaces/default/statefulsets",
        		"pods": "http://host/apis/zcloud.cn/v1/clusters/beijing/namespaces/default/statefulsets/sts123/pods",
        		"remove": "http://host/apis/zcloud.cn/v1/clusters/beijing/namespaces/default/statefulsets/sts123",
        		"self": "http://host/apis/zcloud.cn/v1/clusters/beijing/namespaces/default/statefulsets/sts123",
        		"update": "http://host/apis/zcloud.cn/v1/clusters/beijing/namespaces/default/statefulsets/sts123"
    		}
    	} 
   
  * links说明如下  		
    * 如果资源支持单个资源的get，即资源schema的ResourceMethods中设置了GET，links中就会包含self
    * 如果资源支持所有资源的list， 即资源schema的CollectionMethods中设置了GET，links中就会包含collection
    * 如果资源支持删除操作，即资源schema的ResourceMethods中设置了DELETE，links中就会包含remove
    * 如果资源支持更新操作，即资源schema的ResourceMethods中设置了PUT，links中就会包含update
    * 如果资源有子资源，如statefulset的是pod父资源，links中会包含pod的collection，即pods

# Example
在集群zcloud的default命名空间下创建一个deployment

* 使用admin用户登录服务端
  * Request	
		
		curl POST http://10.0.0.140:1234/apis/zcloud.cn/v1/users/admin?action=login
		{
			"password": "0192309fba8c6f0929b5b00867ebccac9a39e34e"
		}
  * Response
  
		{
			"token": “eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9”
		}
	
* 在default命名空间下创建deployment
  * Request 

		curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
		POST http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments
		{
			"name": "dm123",
			"replicas": 1,
			"containers": [
				{
					"name": "dm123c1",
					"image": "redis"
				}
			]
		}
	
  * Response
	
		{
    		"id": "dm123",
    		"type": "deployment",
    		"links": {
        		"collection": "http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments",
        		"pods": "http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments/dm123/pods",
        		"remove": "http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments/dm123",
        		"self": "http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments/dm123",
        		"update": "http://host/apis/zcloud.cn/v1/clusters/zcloud/namespaces/default/deployments/dm123"
    		},
    		"creationTimestamp": null,
    		"name": "dm123",
    		"replicas": 1,
    		"containers": [
        		{
            		"name": "dm123c1",
            		"image": "redis"
        		}
    		]
    	}
		