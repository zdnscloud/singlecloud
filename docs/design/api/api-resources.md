singlecloud api
======

# Resources

## Application
Collection name is applications, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/applications
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/applications
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"chartName": {
					"required": "true",
					"type": "string"
				},
				"chartVersion": {
					"required": "true",
					"type": "string"
				},
				"configs": {
					"required": "true",
					"type": "json"
				},
				"name": {
					"required": "true",
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"appResources": {
					"elemType": "appResource",
					"type": "array"
				},
				"chartIcon": {
					"type": "string"
				},
				"chartName": {
					"type": "string"
				},
				"chartVersion": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"create",
						"delete",
						"succeed",
						"failed"
					]
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"appResource": {
					"link": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"deployment",
							"daemonset",
							"statefulset",
							"configmap",
							"secret",
							"service",
							"ingress",
							"cronjob",
							"job"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/applications/{application_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Blockdevice
Collection name is blockdevices, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/blockdevices
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Chart
Collection name is charts, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/charts
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/charts/{chart_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"description": {
					"type": "string"
				},
				"icon": {
					"type": "string"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				},
				"versions": {
					"elemType": "chartVersion",
					"type": "array"
				}
			},
			"subResources": {
				"chartVersion": {
					"config": {
						"elemType": "json",
						"type": "array"
					},
					"version": {
						"type": "string"
					}
				}
			}
		}



## Cluster
Collection name is clusters

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"clusterCidr": {
					"required": true,
					"type": "string"
				},
				"clusterDNSServiceIP": {
					"required": true,
					"type": "string"
				},
				"clusterDomain": {
					"required": true,
					"type": "string"
				},
				"clusterUpstreamDNS": {
					"elemType": "string",
					"required": true,
					"type": "array"
				},
				"disablePortCheck": {
					"type": "bool"
				},
				"dockerSocket": {
					"type": "string"
				},
				"ignoreDockerVersion": {
					"type": "bool"
				},
				"kubernetsVersion": {
					"type": "string"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"network": {
					"required": true,
					"type": "zkeConfigNetwork"
				},
				"nodes": {
					"elemType": "zkeConfigNode",
					"required": true,
					"type": "array"
				},
				"privateRegistries": {
					"elemType": "privateRegistry",
					"type": "array"
				},
				"serviceCidr": {
					"required": true,
					"type": "string"
				},
				"singlecloudAddress": {
					"required": true,
					"type": "string"
				},
				"sshKey": {
					"required": true,
					"type": "string"
				},
				"sshPort": {
					"required": true,
					"type": "string"
				},
				"sshUser": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"privateRegistry": {
					"cacert": {
						"type": "string"
					},
					"password": {
						"type": "string"
					},
					"url": {
						"type": "string"
					},
					"user": {
						"type": "string"
					}
				},
				"zkeConfigNetwork": {
					"iface": {
						"type": "string"
					},
					"plugin": {
						"type": "enum",
						"validValues": [
							"flannel",
							"calico"
						]
					}
				},
				"zkeConfigNode": {
					"address": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"roles": {
						"elemType": "enum",
						"required": true,
						"type": "array",
						"validValues": [
							"controlplane",
							"etcd",
							"worker",
							"edge"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"clusterCidr": {
					"type": "string"
				},
				"clusterDNSServiceIP": {
					"type": "string"
				},
				"clusterDomain": {
					"type": "string"
				},
				"clusterUpstreamDNS": {
					"elemType": "string",
					"type": "array"
				},
				"cpu": {
					"type": "int"
				},
				"cpuUsed": {
					"type": "int"
				},
				"cpuUsedRatio": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"disablePortCheck": {
					"type": "bool"
				},
				"dockerSocket": {
					"type": "string"
				},
				"id": {
					"type": "string"
				},
				"ignoreDockerVersion": {
					"type": "bool"
				},
				"kubernetesVersion": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "int"
				},
				"memoryUsed": {
					"type": "int"
				},
				"memoryUsedRatio": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"network": {
					"type": "zkeConfigNetwork"
				},
				"nodeCount": {
					"type": "uint32"
				},
				"nodes": {
					"elemType": "zkeConfigNode",
					"type": "array"
				},
				"option": {
					"type": "zkeConfigOption"
				},
				"pod": {
					"type": "int"
				},
				"podUsed": {
					"type": "int"
				},
				"podUsedRatio": {
					"type": "string"
				},
				"privateRegistries": {
					"elemType": "privateRegistry",
					"type": "array"
				},
				"serviceCidr": {
					"type": "string"
				},
				"singleCloudAddress": {
					"type": "string"
				},
				"sshKey": {
					"type": "string"
				},
				"sshPort": {
					"type": "string"
				},
				"sshUser": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"Running",
						"Unreachable",
						"Creating",
						"Updating",
						"Connecting",
						"Unavailable",
						"Canceling"
					]
				},
				"type": {
					"type": "string"
				},
				"version": {
					"type": "string"
				}
			},
			"subResources": {
				"privateRegistry": {
					"caCert": {
						"type": "string"
					},
					"password": {
						"type": "string"
					},
					"url": {
						"type": "string"
					},
					"user": {
						"type": "string"
					}
				},
				"zkeConfigNetwork": {
					"iface": {
						"type": "string"
					},
					"plugin": {
						"type": "enum",
						"validValues": [
							"flannel",
							"calico"
						]
					}
				},
				"zkeConfigNode": {
					"address": {
						"type": "string"
					},
					"annotations": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"cpu": {
						"type": "int"
					},
					"cpuUsed": {
						"type": "int"
					},
					"cpuUsedRatio": {
						"type": "string"
					},
					"dockerVersion": {
						"type": "string"
					},
					"labels": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"memory": {
						"type": "int"
					},
					"memoryUsed": {
						"type": "int"
					},
					"memoryUsedRatio": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"operatingSystem": {
						"type": "string"
					},
					"operatingSystemImage": {
						"type": "string"
					},
					"pod": {
						"type": "int"
					},
					"podUsed": {
						"type": "int"
					},
					"podUsedRatio": {
						"type": "string"
					},
					"roles": {
						"elemType": "enum",
						"type": "array",
						"validValues": [
							"controlplane",
							"etcd",
							"worker",
							"edge"
						]
					},
					"status": {
						"type": "string"
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"clusterCidr": {
					"type": "string"
				},
				"clusterDNSServiceIP": {
					"type": "string"
				},
				"clusterDomain": {
					"type": "string"
				},
				"clusterUpstreamDNS": {
					"elemType": "string",
					"type": "array"
				},
				"cpu": {
					"type": "int"
				},
				"cpuUsed": {
					"type": "int"
				},
				"cpuUsedRatio": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"disablePortCheck": {
					"type": "bool"
				},
				"dockerSocket": {
					"type": "string"
				},
				"id": {
					"type": "string"
				},
				"ignoreDockerVersion": {
					"type": "bool"
				},
				"kubernetesVersion": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "int"
				},
				"memoryUsed": {
					"type": "int"
				},
				"memoryUsedRatio": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"network": {
					"type": "zkeConfigNetwork"
				},
				"nodeCount": {
					"type": "uint32"
				},
				"nodes": {
					"elemType": "zkeConfigNode",
					"type": "array"
				},
				"option": {
					"type": "zkeConfigOption"
				},
				"pod": {
					"type": "int"
				},
				"podUsed": {
					"type": "int"
				},
				"podUsedRatio": {
					"type": "string"
				},
				"privateRegistries": {
					"elemType": "privateRegistry",
					"type": "array"
				},
				"serviceCidr": {
					"type": "string"
				},
				"singleCloudAddress": {
					"type": "string"
				},
				"sshKey": {
					"type": "string"
				},
				"sshPort": {
					"type": "string"
				},
				"sshUser": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"Running",
						"Unreachable",
						"Creating",
						"Updating",
						"Connecting",
						"Unavailable",
						"Canceling"
					]
				},
				"type": {
					"type": "string"
				},
				"version": {
					"type": "string"
				}
			},
			"subResources": {
				"privateRegistry": {
					"caCert": {
						"type": "string"
					},
					"password": {
						"type": "string"
					},
					"url": {
						"type": "string"
					},
					"user": {
						"type": "string"
					}
				},
				"zkeConfigNetwork": {
					"iface": {
						"type": "string"
					},
					"plugin": {
						"type": "enum",
						"validValues": [
							"flannel",
							"calico"
						]
					}
				},
				"zkeConfigNode": {
					"address": {
						"type": "string"
					},
					"annotations": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"cpu": {
						"type": "int"
					},
					"cpuUsed": {
						"type": "int"
					},
					"cpuUsedRatio": {
						"type": "string"
					},
					"dockerVersion": {
						"type": "string"
					},
					"labels": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"memory": {
						"type": "int"
					},
					"memoryUsed": {
						"type": "int"
					},
					"memoryUsedRatio": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"operatingSystem": {
						"type": "string"
					},
					"operatingSystemImage": {
						"type": "string"
					},
					"pod": {
						"type": "int"
					},
					"podUsed": {
						"type": "int"
					},
					"podUsedRatio": {
						"type": "string"
					},
					"roles": {
						"elemType": "enum",
						"type": "array",
						"validValues": [
							"controlplane",
							"etcd",
							"worker",
							"edge"
						]
					},
					"status": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		null


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"clusterCidr": {
					"type": "string"
				},
				"clusterDNSServiceIP": {
					"type": "string"
				},
				"clusterDomain": {
					"type": "string"
				},
				"clusterUpstreamDNS": {
					"elemType": "string",
					"type": "array"
				},
				"cpu": {
					"type": "int"
				},
				"cpuUsed": {
					"type": "int"
				},
				"cpuUsedRatio": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"disablePortCheck": {
					"type": "bool"
				},
				"dockerSocket": {
					"type": "string"
				},
				"id": {
					"type": "string"
				},
				"ignoreDockerVersion": {
					"type": "bool"
				},
				"kubernetesVersion": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "int"
				},
				"memoryUsed": {
					"type": "int"
				},
				"memoryUsedRatio": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"network": {
					"type": "zkeConfigNetwork"
				},
				"nodeCount": {
					"type": "uint32"
				},
				"nodes": {
					"elemType": "zkeConfigNode",
					"type": "array"
				},
				"option": {
					"type": "zkeConfigOption"
				},
				"pod": {
					"type": "int"
				},
				"podUsed": {
					"type": "int"
				},
				"podUsedRatio": {
					"type": "string"
				},
				"privateRegistries": {
					"elemType": "privateRegistry",
					"type": "array"
				},
				"serviceCidr": {
					"type": "string"
				},
				"singleCloudAddress": {
					"type": "string"
				},
				"sshKey": {
					"type": "string"
				},
				"sshPort": {
					"type": "string"
				},
				"sshUser": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"Running",
						"Unreachable",
						"Creating",
						"Updating",
						"Connecting",
						"Unavailable",
						"Canceling"
					]
				},
				"type": {
					"type": "string"
				},
				"version": {
					"type": "string"
				}
			},
			"subResources": {
				"privateRegistry": {
					"caCert": {
						"type": "string"
					},
					"password": {
						"type": "string"
					},
					"url": {
						"type": "string"
					},
					"user": {
						"type": "string"
					}
				},
				"zkeConfigNetwork": {
					"iface": {
						"type": "string"
					},
					"plugin": {
						"type": "enum",
						"validValues": [
							"flannel",
							"calico"
						]
					}
				},
				"zkeConfigNode": {
					"address": {
						"type": "string"
					},
					"annotations": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"cpu": {
						"type": "int"
					},
					"cpuUsed": {
						"type": "int"
					},
					"cpuUsedRatio": {
						"type": "string"
					},
					"dockerVersion": {
						"type": "string"
					},
					"labels": {
						"keyType": "string",
						"type": "map",
						"valueType": "string"
					},
					"memory": {
						"type": "int"
					},
					"memoryUsed": {
						"type": "int"
					},
					"memoryUsedRatio": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"operatingSystem": {
						"type": "string"
					},
					"operatingSystemImage": {
						"type": "string"
					},
					"pod": {
						"type": "int"
					},
					"podUsed": {
						"type": "int"
					},
					"podUsedRatio": {
						"type": "string"
					},
					"roles": {
						"elemType": "enum",
						"type": "array",
						"validValues": [
							"controlplane",
							"etcd",
							"worker",
							"edge"
						]
					},
					"status": {
						"type": "string"
					}
				}
			}
		}



#### Cancel
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}?action=cancel
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  

#### Getkubeconfig
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}?action=getkubeconfig
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"output": {
				"config": {
					"type": "string"
				},
				"name": {
					"type": "string"
				}
			}
		}



## Configmap
Collection name is configmaps, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/configmaps
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"configs": {
					"elemType": "config",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"config": {
					"data": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"configs": {
					"elemType": "config",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"config": {
					"data": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/configmaps
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/configmaps/{configmap_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"configs": {
					"elemType": "config",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"config": {
					"data": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/configmaps/{configmap_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		null


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"configs": {
					"elemType": "config",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"config": {
					"data": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/configmaps/{configmap_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Cronjob
Collection name is cronjobs, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/cronjobs
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/cronjobs
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"containers": {
					"elemType": "container",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"restartPolicy": {
					"required": true,
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				},
				"schedule": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"restartPolicy": {
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				},
				"schedule": {
					"type": "string"
				},
				"status": {
					"type": "cronJobStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"cronJobStatus": {
					"lastScheduleTime": {
						"type": "date"
					},
					"objectReferences": {
						"elemType": "objectReference",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"objectReference": {
					"apiVersion": {
						"type": "string"
					},
					"fieldPath": {
						"type": "string"
					},
					"kind": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					},
					"resourceVersion": {
						"type": "string"
					},
					"uid": {
						"type": "string"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/cronjobs/{cronjob_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"restartPolicy": {
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				},
				"schedule": {
					"type": "string"
				},
				"status": {
					"type": "cronJobStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"cronJobStatus": {
					"lastScheduleTime": {
						"type": "date"
					},
					"objectReferences": {
						"elemType": "objectReference",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"objectReference": {
					"apiVersion": {
						"type": "string"
					},
					"fieldPath": {
						"type": "string"
					},
					"kind": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					},
					"resourceVersion": {
						"type": "string"
					},
					"uid": {
						"type": "string"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/cronjobs/{cronjob_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Daemonset
Collection name is daemonsets, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"status": {
					"type": "daemonsetStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"daemonsetCondition": {
					"lastTransitionTime": {
						"type": "date"
					},
					"message": {
						"type": "string"
					},
					"reason": {
						"type": "string"
					},
					"status": {
						"type": "enum",
						"validValues": [
							"True",
							"False",
							"Unknown"
						]
					},
					"type": {
						"type": "string"
					}
				},
				"daemonsetStatus": {
					"collisionCount": {
						"type": "int32"
					},
					"conditions": {
						"type": "daemonsetCondition"
					},
					"currentNumberScheduled": {
						"type": "int32"
					},
					"desiredNumberScheduled": {
						"type": "int32"
					},
					"numberAvailable": {
						"type": "int32"
					},
					"numberMisscheduled": {
						"type": "int32"
					},
					"numberReady": {
						"type": "int32"
					},
					"numberUnavailable": {
						"type": "int32"
					},
					"observedGeneration": {
						"type": "int64"
					},
					"updatedNumberScheduled": {
						"type": "int32"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets/{daemonset_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"status": {
					"type": "daemonsetStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"daemonsetCondition": {
					"lastTransitionTime": {
						"type": "date"
					},
					"message": {
						"type": "string"
					},
					"reason": {
						"type": "string"
					},
					"status": {
						"type": "enum",
						"validValues": [
							"True",
							"False",
							"Unknown"
						]
					},
					"type": {
						"type": "string"
					}
				},
				"daemonsetStatus": {
					"collisionCount": {
						"type": "int32"
					},
					"conditions": {
						"type": "daemonsetCondition"
					},
					"currentNumberScheduled": {
						"type": "int32"
					},
					"desiredNumberScheduled": {
						"type": "int32"
					},
					"numberAvailable": {
						"type": "int32"
					},
					"numberMisscheduled": {
						"type": "int32"
					},
					"numberReady": {
						"type": "int32"
					},
					"numberUnavailable": {
						"type": "int32"
					},
					"observedGeneration": {
						"type": "int64"
					},
					"updatedNumberScheduled": {
						"type": "int32"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets/{daemonset_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### History
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets/{daemonset_id}?action=history
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"output": {
				"history": {
					"elemType": "versionInfo",
					"type": "array"
				}
			}
		}



#### SetImage
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets/{daemonset_id}?action=setImage
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"images": {
					"elemType": "containerImage",
					"type": "array"
				},
				"reason": {
					"type": "string"
				}
			},
			"subResources": {
				"containerImage": {
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

#### Rollback
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/daemonsets/{daemonset_id}?action=rollback
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"reason": {
					"type": "string"
				},
				"version": {
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

## Deployment
Collection name is deployments, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"required": true,
					"type": "int"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "containerPort",
						"type": "array"
					},
					"image": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"containerPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"required": true,
						"type": "int"
					},
					"protocol": {
						"required": true,
						"type": "enum",
						"validValues": [
							"TCP",
							"UDP"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "containerPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"containerPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "containerPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"containerPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"replicas": {
					"required": true,
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "containerPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"containerPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### History
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}?action=history
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"output": {
				"history": {
					"elemType": "versionInfo",
					"type": "array"
				}
			}
		}



#### SetImage
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}?action=setImage
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"images": {
					"elemType": "containerImage",
					"type": "array"
				},
				"reason": {
					"type": "string"
				}
			},
			"subResources": {
				"containerImage": {
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

#### Rollback
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}?action=rollback
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"reason": {
					"type": "string"
				},
				"version": {
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

## Ingress
Collection name is ingresses, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/ingresses
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"name": {
					"required": true,
					"type": "string"
				},
				"rules": {
					"elemType": "ingressRule",
					"required": true,
					"type": "array"
				}
			},
			"subResources": {
				"ingressRule": {
					"host": {
						"required": true,
						"type": "string"
					},
					"path": {
						"required": true,
						"type": "string"
					},
					"serviceName": {
						"required": true,
						"type": "string"
					},
					"servicePort": {
						"required": true,
						"type": "int"
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"rules": {
					"elemType": "ingressRule",
					"type": "array"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"ingressRule": {
					"host": {
						"type": "string"
					},
					"path": {
						"type": "string"
					},
					"serviceName": {
						"type": "string"
					},
					"servicePort": {
						"type": "int"
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/ingresses
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/ingresses/{ingress_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"rules": {
					"elemType": "ingressRule",
					"type": "array"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"ingressRule": {
					"host": {
						"type": "string"
					},
					"path": {
						"type": "string"
					},
					"serviceName": {
						"type": "string"
					},
					"servicePort": {
						"type": "int"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/ingresses/{ingress_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Innerservice
Collection name is innerservices, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/innerservices
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Job
Collection name is jobs, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/jobs
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/jobs
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"containers": {
					"elemType": "container",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"restartPolicy": {
					"required": true,
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"restartPolicy": {
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				},
				"status": {
					"type": "jobStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"jobCondition": {
					"lastProbeTime": {
						"type": "date"
					},
					"lastTransitionTime": {
						"type": "date"
					},
					"message": {
						"type": "string"
					},
					"reason": {
						"type": "string"
					},
					"status": {
						"type": "enum",
						"validValues": [
							"True",
							"False",
							"Unknown"
						]
					},
					"type": {
						"type": "enum",
						"validValues": [
							"Complete",
							"Failed"
						]
					}
				},
				"jobStatus": {
					"active": {
						"type": "int32"
					},
					"completionTime": {
						"type": "date"
					},
					"failed": {
						"type": "int32"
					},
					"jobConditions": {
						"elemType": "jobCondition",
						"type": "array"
					},
					"startTime": {
						"type": "date"
					},
					"succeeded": {
						"type": "int32"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/jobs/{job_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"restartPolicy": {
					"type": "enum",
					"validValues": [
						"OnFailure",
						"Never"
					]
				},
				"status": {
					"type": "jobStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"jobCondition": {
					"lastProbeTime": {
						"type": "date"
					},
					"lastTransitionTime": {
						"type": "date"
					},
					"message": {
						"type": "string"
					},
					"reason": {
						"type": "string"
					},
					"status": {
						"type": "enum",
						"validValues": [
							"True",
							"False",
							"Unknown"
						]
					},
					"type": {
						"type": "enum",
						"validValues": [
							"Complete",
							"Failed"
						]
					}
				},
				"jobStatus": {
					"active": {
						"type": "int32"
					},
					"completionTime": {
						"type": "date"
					},
					"failed": {
						"type": "int32"
					},
					"jobConditions": {
						"elemType": "jobCondition",
						"type": "array"
					},
					"startTime": {
						"type": "date"
					},
					"succeeded": {
						"type": "int32"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/jobs/{job_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Limitrange
Collection name is limitranges, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/limitranges
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/limitranges
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"max": {
					"keyType": "resourceName",
					"required": true,
					"type": "map",
					"valueType": "string"
				},
				"min": {
					"keyType": "resourceName",
					"required": true,
					"type": "map",
					"valueType": "string"
				},
				"name": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"cpu",
						"memory"
					]
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"max": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"min": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"cpu",
						"memory"
					]
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/limitranges/{limitrange_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"max": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"min": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"cpu",
						"memory"
					]
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/limitranges/{limitrange_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Monitor
Collection name is monitors, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/monitors
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/monitors
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"adminPassword": {
					"type": "string"
				},
				"ingressDomain": {
					"type": "string"
				},
				"prometheusRetention": {
					"type": "int"
				},
				"scrapeInterval": {
					"type": "int"
				},
				"storageClass": {
					"type": "enum",
					"validValues": [
						"cephfs",
						"lvm"
					]
				},
				"storageSize": {
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"adminPassword": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"ingressDomain": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"prometheusRetention": {
					"type": "int"
				},
				"redirectUrl": {
					"type": "string"
				},
				"scrapeInterval": {
					"type": "int"
				},
				"storageClass": {
					"type": "enum",
					"validValues": [
						"cephfs",
						"lvm"
					]
				},
				"storageSize": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/monitors/{monitor_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Namespace
Collection name is namespaces, its parents is cluster

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"name": {
					"required": true,
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Node
Collection name is nodes, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/nodes
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/nodes/{node_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"address": {
					"type": "string"
				},
				"annotations": {
					"keyType": "string",
					"type": "map",
					"valueType": "string"
				},
				"cpu": {
					"type": "int"
				},
				"cpuUsed": {
					"type": "int"
				},
				"cpuUsedRatio": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"dockerVersion": {
					"type": "string"
				},
				"id": {
					"type": "string"
				},
				"labels": {
					"keyType": "string",
					"type": "map",
					"valueType": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "int"
				},
				"memoryUsed": {
					"type": "int"
				},
				"memoryUsedRatio": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"operatingSystem": {
					"type": "string"
				},
				"operatingSystemImage": {
					"type": "string"
				},
				"pod": {
					"type": "int"
				},
				"podUsed": {
					"type": "int"
				},
				"podUsedRatio": {
					"type": "string"
				},
				"roles": {
					"elemType": "string",
					"type": "array"
				},
				"status": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			}
		}



## Nodenetwork
Collection name is nodenetworks, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/nodenetworks
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Outerservice
Collection name is outerservices, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/outerservices
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Persistentvolume
Collection name is persistentvolumes, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/persistentvolumes
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/persistentvolumes/{persistentvolume_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"claimRef": {
					"type": "claimRef"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"Pending",
						"Bound",
						"Available",
						"Released",
						"Failed"
					]
				},
				"storageClassName": {
					"type": "string"
				},
				"storageSize": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"claimRef": {
					"kind": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/persistentvolumes/{persistentvolume_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Persistentvolumeclaim
Collection name is persistentvolumeclaims, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/persistentvolumeclaims
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/persistentvolumeclaims/{persistentvolumeclaim_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"actualStorageSize": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"namespace": {
					"type": "string"
				},
				"requestStorageSize": {
					"type": "string"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"Pending",
						"Bound",
						"Lost"
					]
				},
				"storageClassName": {
					"type": "string"
				},
				"type": {
					"type": "string"
				},
				"volumeName": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/persistentvolumeclaims/{persistentvolumeclaim_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Pod
Collection name is pods, its parents is deployment daemonset statefulset job cronjob

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}/pods
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/deployments/{deployment_id}/pods/{pod_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"nodeName": {
					"type": "string"
				},
				"status": {
					"type": "podStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"containerState": {
					"containerID": {
						"type": "string"
					},
					"exitCode": {
						"type": "int32"
					},
					"finishedAt": {
						"type": "date"
					},
					"message": {
						"type": "string"
					},
					"reason": {
						"type": "string"
					},
					"startedAt": {
						"type": "date"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"waiting",
							"running",
							"terminated"
						]
					}
				},
				"containerStatus": {
					"containerID": {
						"type": "string"
					},
					"image": {
						"type": "string"
					},
					"imageID": {
						"type": "string"
					},
					"lastState": {
						"type": "containerState"
					},
					"name": {
						"type": "string"
					},
					"ready": {
						"type": "bool"
					},
					"restartCount": {
						"type": "int32"
					},
					"state": {
						"type": "containerState"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"podCondition": {
					"lastProbeTime": {
						"type": "date"
					},
					"lastTransitionTime": {
						"type": "date"
					},
					"status": {
						"type": "enum",
						"validValues": [
							"True",
							"False",
							"Unknown"
						]
					},
					"type": {
						"type": "enum",
						"validValues": [
							"PodScheduled",
							"Ready",
							"Initialized",
							"ContainersReady"
						]
					}
				},
				"podStatus": {
					"containerStatuses": {
						"elemType": "containerStatus",
						"type": "array"
					},
					"hostIP": {
						"type": "string"
					},
					"phase": {
						"type": "enum",
						"validValues": [
							"Pending",
							"Running",
							"Succeeded",
							"Failed",
							"Unknown"
						]
					},
					"podConditions": {
						"elemType": "podCondition",
						"type": "array"
					},
					"podIP": {
						"type": "string"
					},
					"startTime": {
						"type": "date"
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



## Podnetwork
Collection name is podnetworks, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/podnetworks
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Registry
Collection name is registries

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/registries
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/registries
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"adminPassword": {
					"type": "string"
				},
				"cluster": {
					"required": true,
					"type": "string"
				},
				"ingressDomain": {
					"type": "string"
				},
				"storageClass": {
					"type": "enum",
					"validValues": [
						"cephfs",
						"lvm"
					]
				},
				"storageSize": {
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"adminPassword": {
					"type": "string"
				},
				"cluster": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"ingressDomain": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"redirectUrl": {
					"type": "string"
				},
				"storageClass": {
					"type": "enum",
					"validValues": [
						"cephfs",
						"lvm"
					]
				},
				"storageSize": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/registries/{registry_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Resourcequota
Collection name is resourcequotas, its parents is namespace

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/resourcequotas
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/resourcequotas
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"limits": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"name": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"requests.cpu",
						"requests.memory",
						"limits.cpu",
						"limits.memory",
						"requests.storage"
					]
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"limits": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"status": {
					"type": "resourceQuotaStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"requests.cpu",
						"requests.memory",
						"limits.cpu",
						"limits.memory",
						"requests.storage"
					]
				},
				"resourceQuotaStatus": {
					"limits": {
						"type": "resourceList"
					},
					"used": {
						"type": "resourceList"
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/resourcequotas/{resourcequota_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"limits": {
					"keyType": "resourceName",
					"type": "map",
					"valueType": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"status": {
					"type": "resourceQuotaStatus"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"resourceName": {
					"type": "enum",
					"validValues": [
						"requests.cpu",
						"requests.memory",
						"limits.cpu",
						"limits.memory",
						"requests.storage"
					]
				},
				"resourceQuotaStatus": {
					"limits": {
						"type": "resourceList"
					},
					"used": {
						"type": "resourceList"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/resourcequotas/{resourcequota_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Secret
Collection name is secrets, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/secrets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"data": {
					"elemType": "secretData",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				}
			},
			"subResources": {
				"secretData": {
					"key": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"data": {
					"elemType": "secretData",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"secretData": {
					"key": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/secrets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/secrets/{secret_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"data": {
					"elemType": "secretData",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"secretData": {
					"key": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/secrets/{secret_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		null


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"data": {
					"elemType": "secretData",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"secretData": {
					"key": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/secrets/{secret_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Service
Collection name is services, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/services
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"exposedPorts": {
					"elemType": "servicePort",
					"required": true,
					"type": "array"
				},
				"headless": {
					"type": "bool"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"serviceType": {
					"required": true,
					"type": "enum",
					"validValues": [
						"clusterip",
						"nodeport"
					]
				}
			},
			"subResources": {
				"servicePort": {
					"name": {
						"required": true,
						"type": "string"
					},
					"port": {
						"required": true,
						"type": "int"
					},
					"protocol": {
						"required": true,
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					},
					"targetPort": {
						"required": true,
						"type": "int"
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"clusterIP": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"exposedPorts": {
					"elemType": "servicePort",
					"type": "array"
				},
				"headless": {
					"type": "bool"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"serviceType": {
					"type": "enum",
					"validValues": [
						"clusterip",
						"nodeport"
					]
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"servicePort": {
					"name": {
						"type": "string"
					},
					"nodePort": {
						"type": "int"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					},
					"targetPort": {
						"type": "int"
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/services
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/services/{service_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"clusterIP": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"exposedPorts": {
					"elemType": "servicePort",
					"type": "array"
				},
				"headless": {
					"type": "bool"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"serviceType": {
					"type": "enum",
					"validValues": [
						"clusterip",
						"nodeport"
					]
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"servicePort": {
					"name": {
						"type": "string"
					},
					"nodePort": {
						"type": "int"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					},
					"targetPort": {
						"type": "int"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/services/{service_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Servicenetwork
Collection name is servicenetworks, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/servicenetworks
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Statefulset
Collection name is statefulsets, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"required": true,
					"type": "int"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"required": true,
						"type": "string"
					},
					"name": {
						"required": true,
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"required": true,
						"type": "int"
					},
					"protocol": {
						"required": true,
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		null


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"advancedOptions": {
					"type": "advancedOptions"
				},
				"containers": {
					"elemType": "container",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"persistentVolumes": {
					"elemType": "persistentVolume",
					"type": "array"
				},
				"replicas": {
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"advancedOptions": {
					"deletePVsWhenDeleteWorkload": {
						"type": "bool"
					},
					"exposedMetric": {
						"type": "exposedMetric"
					},
					"reloadWhenConfigChange": {
						"type": "bool"
					}
				},
				"container": {
					"args": {
						"elemType": "string",
						"type": "array"
					},
					"command": {
						"elemType": "string",
						"type": "array"
					},
					"env": {
						"elemType": "envVar",
						"type": "array"
					},
					"exposedPorts": {
						"elemType": "deploymentPort",
						"type": "array"
					},
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"volumes": {
						"elemType": "volume",
						"type": "array"
					}
				},
				"deploymentPort": {
					"name": {
						"type": "string"
					},
					"port": {
						"type": "int"
					},
					"protocol": {
						"type": "enum",
						"validValues": [
							"tcp",
							"udp"
						]
					}
				},
				"envVar": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					}
				},
				"exposedMetric": {
					"path": {
						"type": "string"
					},
					"port": {
						"type": "int"
					}
				},
				"persistentVolume": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"storageClassName": {
						"type": "enum",
						"validValues": [
							"lvm",
							"nfs",
							"temporary",
							"cephfs"
						]
					}
				},
				"volume": {
					"mountPath": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"type": {
						"type": "enum",
						"validValues": [
							"configmap",
							"secret",
							"persistentVolume"
						]
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### History
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}?action=history
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"output": {
				"history": {
					"elemType": "versionInfo",
					"type": "array"
				}
			}
		}



#### SetImage
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}?action=setImage
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"images": {
					"elemType": "containerImage",
					"type": "array"
				},
				"reason": {
					"type": "string"
				}
			},
			"subResources": {
				"containerImage": {
					"image": {
						"type": "string"
					},
					"name": {
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

#### Rollback
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/statefulsets/{statefulset_id}?action=rollback
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"reason": {
					"type": "string"
				},
				"version": {
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

## Storageclass
Collection name is storageclasses, its parents is cluster

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclasses
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



## Storagecluster
Collection name is storageclusters, its parents is cluster

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclusters
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"hosts": {
					"elemType": "string",
					"required": true,
					"type": "array"
				},
				"name": {
					"required": true,
					"type": "string"
				},
				"storagetype": {
					"required": true,
					"type": "enum",
					"validValues": [
						"lvm",
						"ceph"
					]
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"config": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"freeDevs": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"freeSize": {
					"type": "string"
				},
				"hosts": {
					"elemType": "string",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"nodes": {
					"elemType": "storagenode",
					"type": "array"
				},
				"phase": {
					"type": "string"
				},
				"pvs": {
					"elemType": "storagepv",
					"type": "array"
				},
				"size": {
					"type": "string"
				},
				"storageType": {
					"type": "enum",
					"validValues": [
						"lvm",
						"ceph"
					]
				},
				"type": {
					"type": "string"
				},
				"usedSize": {
					"type": "string"
				}
			},
			"subResources": {
				"storagecfg": {
					"blockDevices": {
						"elemType": "storagedev",
						"type": "array"
					},
					"nodeName": {
						"type": "string"
					}
				},
				"storagedev": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					}
				},
				"storagenode": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"stat": {
						"type": "bool"
					},
					"usedSize": {
						"type": "string"
					}
				},
				"storagepod": {
					"name": {
						"type": "string"
					}
				},
				"storagepv": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"node": {
						"type": "string"
					},
					"pods": {
						"elemType": "storagepod",
						"type": "array"
					},
					"size": {
						"type": "string"
					},
					"usedSize": {
						"type": "string"
					}
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclusters
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclusters/{storagecluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"config": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"freeDevs": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"freeSize": {
					"type": "string"
				},
				"hosts": {
					"elemType": "string",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"nodes": {
					"elemType": "storagenode",
					"type": "array"
				},
				"phase": {
					"type": "string"
				},
				"pvs": {
					"elemType": "storagepv",
					"type": "array"
				},
				"size": {
					"type": "string"
				},
				"storageType": {
					"type": "enum",
					"validValues": [
						"lvm",
						"ceph"
					]
				},
				"type": {
					"type": "string"
				},
				"usedSize": {
					"type": "string"
				}
			},
			"subResources": {
				"storagecfg": {
					"blockDevices": {
						"elemType": "storagedev",
						"type": "array"
					},
					"nodeName": {
						"type": "string"
					}
				},
				"storagedev": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					}
				},
				"storagenode": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"stat": {
						"type": "bool"
					},
					"usedSize": {
						"type": "string"
					}
				},
				"storagepod": {
					"name": {
						"type": "string"
					}
				},
				"storagepv": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"node": {
						"type": "string"
					},
					"pods": {
						"elemType": "storagepod",
						"type": "array"
					},
					"size": {
						"type": "string"
					},
					"usedSize": {
						"type": "string"
					}
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclusters/{storagecluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"hosts": {
					"elemType": "string",
					"required": true,
					"type": "array"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"config": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"freeDevs": {
					"elemType": "storagecfg",
					"type": "array"
				},
				"freeSize": {
					"type": "string"
				},
				"hosts": {
					"elemType": "string",
					"type": "array"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"nodes": {
					"elemType": "storagenode",
					"type": "array"
				},
				"phase": {
					"type": "string"
				},
				"pvs": {
					"elemType": "storagepv",
					"type": "array"
				},
				"size": {
					"type": "string"
				},
				"storageType": {
					"type": "enum",
					"validValues": [
						"lvm",
						"ceph"
					]
				},
				"type": {
					"type": "string"
				},
				"usedSize": {
					"type": "string"
				}
			},
			"subResources": {
				"storagecfg": {
					"blockDevices": {
						"elemType": "storagedev",
						"type": "array"
					},
					"nodeName": {
						"type": "string"
					}
				},
				"storagedev": {
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					}
				},
				"storagenode": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"size": {
						"type": "string"
					},
					"stat": {
						"type": "bool"
					},
					"usedSize": {
						"type": "string"
					}
				},
				"storagepod": {
					"name": {
						"type": "string"
					}
				},
				"storagepv": {
					"freeSize": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"node": {
						"type": "string"
					},
					"pods": {
						"elemType": "storagepod",
						"type": "array"
					},
					"size": {
						"type": "string"
					},
					"usedSize": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/storageclusters/{storagecluster_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## Udpingress
Collection name is udpingresses, its parents is namespace

#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/udpingresses
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"port": {
					"required": true,
					"type": "int"
				},
				"serviceName": {
					"required": true,
					"type": "string"
				},
				"servicePort": {
					"required": true,
					"type": "int"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"port": {
					"required": true,
					"type": "int"
				},
				"serviceName": {
					"required": true,
					"type": "string"
				},
				"servicePort": {
					"required": true,
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/udpingresses
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/udpingresses/{udpingress_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"port": {
					"required": true,
					"type": "int"
				},
				"serviceName": {
					"required": true,
					"type": "string"
				},
				"servicePort": {
					"required": true,
					"type": "int"
				},
				"type": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/clusters/{cluster_id}/namespaces/{namespace_id}/udpingresses/{udpingress_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

## User
Collection name is users

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/users
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/users
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"name": {
					"required": true,
					"type": "string"
				},
				"password": {
					"required": true,
					"type": "string"
				},
				"projects": {
					"elemType": "project",
					"type": "array"
				}
			},
			"subResources": {
				"project": {
					"cluster": {
						"required": true,
						"type": "string"
					},
					"namespace": {
						"required": true,
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"password": {
					"type": "string"
				},
				"projects": {
					"elemType": "project",
					"type": "array"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"project": {
					"cluster": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					}
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/users/{user_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"password": {
					"type": "string"
				},
				"projects": {
					"elemType": "project",
					"type": "array"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"project": {
					"cluster": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					}
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/users/{user_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/users/{user_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"password": {
					"required": true,
					"type": "string"
				},
				"projects": {
					"elemType": "project",
					"type": "array"
				}
			},
			"subResources": {
				"project": {
					"cluster": {
						"required": true,
						"type": "string"
					},
					"namespace": {
						"required": true,
						"type": "string"
					}
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"name": {
					"type": "string"
				},
				"password": {
					"type": "string"
				},
				"projects": {
					"elemType": "project",
					"type": "array"
				},
				"type": {
					"type": "string"
				}
			},
			"subResources": {
				"project": {
					"cluster": {
						"type": "string"
					},
					"namespace": {
						"type": "string"
					}
				}
			}
		}



#### Login
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/users/{user_id}?action=login
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"password": {
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"output": {
				"token": {
					"type": "string"
				}
			}
		}



#### ResetPassword
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/users/{user_id}?action=resetPassword
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"newPassword": {
					"type": "string"
				},
				"oldPassword": {
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

## Userquota
Collection name is userquotas

#### List
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/userquotas
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body Parameters

		{
			"type": {"type": "string"},
			"resourceType": {"type": "string"},
			"links": {"type": "map"},
			"data": {"type": "array", "elemType": "Resource"},
		}



#### Create
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/userquotas
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"cpu": {
					"required": "true",
					"type": "string"
				},
				"memory": {
					"required": "true",
					"type": "string"
				},
				"namespace": {
					"required": "true",
					"type": "string"
				},
				"purpose": {
					"required": "true",
					"type": "string"
				},
				"requestor": {
					"type": "string"
				},
				"storage": {
					"required": "true",
					"type": "string"
				},
				"telephone": {
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		201 Created	
  
  * Body

		{
			"resourceFields": {
				"clusterName": {
					"type": "string"
				},
				"cpu": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"namespace": {
					"type": "string"
				},
				"purpose": {
					"type": "string"
				},
				"rejectionReason": {
					"type": "string"
				},
				"requestType": {
					"type": "enum",
					"validValues": [
						"create",
						"update"
					]
				},
				"requestor": {
					"type": "string"
				},
				"responseTimestamp": {
					"type": "date"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"processing",
						"approval",
						"rejection"
					]
				},
				"storage": {
					"type": "string"
				},
				"telephone": {
					"type": "string"
				},
				"type": {
					"type": "string"
				},
				"userName": {
					"type": "string"
				}
			}
		}



#### Get
* Request
  * Http Request
		
		GET /apis/zcloud.cn/v1/userquotas/{userquota_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"clusterName": {
					"type": "string"
				},
				"cpu": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"namespace": {
					"type": "string"
				},
				"purpose": {
					"type": "string"
				},
				"rejectionReason": {
					"type": "string"
				},
				"requestType": {
					"type": "enum",
					"validValues": [
						"create",
						"update"
					]
				},
				"requestor": {
					"type": "string"
				},
				"responseTimestamp": {
					"type": "date"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"processing",
						"approval",
						"rejection"
					]
				},
				"storage": {
					"type": "string"
				},
				"telephone": {
					"type": "string"
				},
				"type": {
					"type": "string"
				},
				"userName": {
					"type": "string"
				}
			}
		}



#### Update
* Request
  * Http Request
		
		PUT /apis/zcloud.cn/v1/userquotas/{userquota_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"fields": {
				"clusterName": {
					"required": "true",
					"type": "string"
				},
				"cpu": {
					"required": "true",
					"type": "string"
				},
				"memory": {
					"required": "true",
					"type": "string"
				},
				"namespace": {
					"required": "true",
					"type": "string"
				},
				"purpose": {
					"required": "true",
					"type": "string"
				},
				"requestor": {
					"type": "string"
				},
				"storage": {
					"required": "true",
					"type": "string"
				},
				"telephone": {
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
  * Body

		{
			"resourceFields": {
				"clusterName": {
					"type": "string"
				},
				"cpu": {
					"type": "string"
				},
				"creationTimestamp": {
					"type": "time"
				},
				"id": {
					"type": "string"
				},
				"links": {
					"type": "map"
				},
				"memory": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"namespace": {
					"type": "string"
				},
				"purpose": {
					"type": "string"
				},
				"rejectionReason": {
					"type": "string"
				},
				"requestType": {
					"type": "enum",
					"validValues": [
						"create",
						"update"
					]
				},
				"requestor": {
					"type": "string"
				},
				"responseTimestamp": {
					"type": "date"
				},
				"status": {
					"type": "enum",
					"validValues": [
						"processing",
						"approval",
						"rejection"
					]
				},
				"storage": {
					"type": "string"
				},
				"telephone": {
					"type": "string"
				},
				"type": {
					"type": "string"
				},
				"userName": {
					"type": "string"
				}
			}
		}



#### Delete
* Request
  * Http Request
		
		DELETE /apis/zcloud.cn/v1/userquotas/{userquota_id}
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
* Response
  * Status Code

		204 No Content	
  

#### Approval
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/userquotas/{userquota_id}?action=approval
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"clusterName": {
					"required": "true",
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  

#### Reject
* Request
  * Http Request
		
		POST /apis/zcloud.cn/v1/userquotas/{userquota_id}?action=reject
		Authorization Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
  
  * Body Parameters

		{
			"input": {
				"reason": {
					"required": "true",
					"type": "string"
				}
			}
		}


* Response
  * Status Code

		200 OK	
  
