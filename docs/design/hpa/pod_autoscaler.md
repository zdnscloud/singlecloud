# Pod AutoScaler
# 概要
通过配置HorizontalPodAutoscaler（简称HPA）资源，以资源（包含cpu和memory）或者自定义metric为指标，对workload的pod个数进行自动扩缩

# 详细设计
## HPA资源
HPA为namespace下的子资源

	type HorizontalPodAutoscaler struct {
    	Name                  string                        
    	ScaleTargetKind       ScaleTargetKind               
    	ScaleTargetName       string
    	MinReplicas           int
    	MaxReplicas           int                              
    	ResourceMetrics       []ResourceMetricSpec
    	CustomMetrics         []CustomMetricSpec                   
    	Status                HorizontalPodAutoscalerStatus  
	}

	type ResourceMetricSpec struct {
    	ResourceName       ResourceName      
    	TargetType         MetricTargetType  
    	AverageValue       string  
    	AverageUtilization int
	}
	
	type CustomMetricSpec struct {
    	MetricName   string 
    	Labels       map[string]string            
    	AverageValue string  
	}
	
## 配置
	
* ScaleTargetKind支持deployment和statefulset，表明HPA对哪一种workload进行配置，即通过ScaleTargetKind和ScaleTargetName字段定位一个workload
* MinReplicas和MaxReplicas表示这个workload的最小和最大replicas值
* ResourceMetrics为HPA基础资源指标的配置，通过指定ResourceName表示以哪种资源为指标，ResourceName支持cpu和memory，TargetType支持Utilization和AverageValue
* CustomMetrics是以自定义metric为指标，通过指定MetricName以及Labels来确定采用哪个自定义metric，只支持AverageValue
* 无论HPA如何动态调整workload的pods个数，所调整的pods个数都不会超过MinReplicas和MaxReplicas的范围。

## 配置范例

	"name": "hpa-vg1-vanguard",
    "scaleTargetKind": "deployment",
    "scaleTargetName": "vg1-vanguard",
    "minReplicas": 1,
    "maxReplicas": 5,
    "customMetrics":[
    	{
    		"metricName": "zdns_vanguard_qps_by_view",
    		"labels": {
    			"module": "server",
    			"view": "default"
    		},
    		"averageValue": "2000"
    	}
    ]
    "resourceMetrics": [
    	{
    		"ResourceName": "cpu",
    		"targetType": "Utilization",
    		"averageUtilization": 50
    	},
    	{
    		"ResourceName": "memory",
    		"targetType": "AverageValue",
    		"averageValue": "256Mi"
    	}
    ]
    
## 自定义指标逻辑
* prometheus采集的应用程序的metric，可以通过prometheus-adapter部署的自定义api server（custom.metrics.k8s.io）获取，prometheus-adapter通过配置其configmap确定采集哪些自定义metric
* 配置customMetrics，通过选择metric name极其对应的labels确定使用哪一个自定义metric，然后修改prometheus-adapter的configmap，重启prometheus-adapter pod
* 由于prometheus-adapter的configmap是针对所有namespace的，所以添加configmap配置时，使用metric.Name_labels哈希值前12位_namespace_hpa.Name作为configmap.Data中rule.Name.As的值，来区分不同namespace的hpa所采用的metric，prometheus-adapter的configmap配置范例如下：

		- seriesQuery: '{__name__=~"zdns_vanguard_qps_by_view",kubernetes_pod_name!="",kubernetes_namespace="default"}'
		  seriesFilters: [］
		  resources:
    		overrides:
				kubernetes_namespace:
        			resource: namespace
        		kubernetes_pod_name:
        			resource: pod
          name:
    		matches: ^zdns_vanguard_qps_by_view$
    		as: zdns_vanguard_qps_by_view_820193bd2d60_default_vg
    	  metricsQuery: sum(zdns_vanguard_qps_by_view{module="server",view="default"}) by (<<.GroupBy>>)

* 使用name.as的名字创建hpa，当prometheus-adapter的pod重启完成，就可以从custom api server获取到对应metric的值

## 期待副本数计算
* 从api server获取workload的metrics，每个pod的metrics为pod中所有container的metrics总量，但是获取到的metrics并不一定包含workload的所有pods，如zdns_vanguard_qps，如果流量从来没有到过某个pod，那么api server是拿不到这个pod的zdns_vanguard_qps
   * 如果指标类型为资源，从metrics.k8s.io获取metrics
   * 如果指标类型为自定义指标，从custom.metrics.k8s.io获取自定义metrics  
* 获取workload所有pods，计算有效pod数readyPodCount，以下三种pods不计数：
  * 有 DeletionTimestamp 或者 pod.Status.Phase为Failed的pod
  * 没有在metrics里面的pods（missingPods）
  * resourceName＝cpu且不ready的pods（ignorePods）
* 把metrics中ignorePods的metric移除
* 使用metrics计算指标值及使用比率，后面会根据使用比率计算期待副本数
  * targetType是平均值，targetAverage为配置的平均值的值
    * 当前平均值currentAverage ＝ metrics累加总量／metrics个数
    * 使用比率为 usageRatio = currentAverage / targetAverage
  * targetType是平均利用率，targetUtilization为配置的平均利用率的值
    * 计算请求总值，即累加每个container的resources.requests对应的资源请求量
    * 当前平均利用率currentUtilization ＝ meitrics累加总量 ／ 请求总量 
    * 使用比率 usageRatio ＝ currentUtilization ／ targetUtilization
* 如果metrics包含了所有pods（没有misssingPods），并且所有pods都ready（没有ignorePods）或者需要缩容（usageRatio < 1.0），则可以进行期待副本数desiredReplicas的计算，对 1 - usageRatio 取绝对值
    * 如果绝对值小于0.1，表明改变不大，则期待副本数desiredReplicas = currentReplicas
    * 如果绝对值大于0.1，期待副本数desiredReplicas 为 usageRatio * readyPodCount的向上取整
* 重新计算使用比率和期待副本数
  * 如果有pods没有metrics中，即有missingPods 
    * 如果需要缩容（usageRatio < 1.0），将missingPods以metric ＝ targetAverage／targetUtilization 加入metrics
    * 如果需要扩容（usageRatio > 1.0），将missingPods以metric ＝ 0 加入 metrics中
  * 如果需要扩容并且有不ready的pods，将ignorePods以metric ＝ 0 加入 metrics中
  * 使用新的metrics再次计算使用比率newUsageRatio，对 1 - newUsageRatio 取绝对值
    * 满足以下三种情况之一，期待副本数desiredReplicas = currentReplicas
      * 绝对值小于0.1
      * usageRatio < 1.0 且 newUsageRatio > 1.0
      * usageRatio > 1.0 且 newUsageRatio < 1.0
    * 期待副本数desiredReplicas 为 usageRatio * metrics个数的向上取整
* 如果workload在进行缩容，那么距离上次变更的时间差必须大于controller manager 参数 horizontal-pod-autoscaler-downscale-stabilization配置置的时间，才可以改变desiredReplicas， 否则还是使用原来的desiredReplicas 
* 由于k8s限制pods增长过快，设置了最大增长量scaleUpLimit，即在当前副本数*2 与 4 中取最大值
  * 如果hpa.MaxReplicas大于scaleUpLimit，那么此次允许的最大副本数maximumAllowedReplicas为scaleUpLimit
  * 如果hpa.MaxReplicas小于scaleUpLimit，那么此次允许的最大副本数maximumAllowedReplicas为hpa.MaxReplicas
* 如果desiredReplicas的值不在hpa.MinReplicas与maximumAllowedReplicas范围内
  * 如果desiredReplicas小于hpa.MinReplicas，则 desiredReplicas ＝ hpa.MinReplicas
  * 如果desiredReplicas大于maximumAllowedReplicas，则 desiredReplicas ＝ maximumAllowedReplicas
* 最终计算出来的期待副本数为desiredReplicas
