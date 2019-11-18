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
    	AverageValue string  
	}
	
## 配置
	
* ScaleTargetKind支持deployment和statefulset，表明HPA对哪一种workload进行配置，即通过ScaleTargetKind和ScaleTargetName字段定位一个workload
* MinReplicas和MaxReplicas表示这个workload的最小和最大replicas值
* ResourceMetrics为HPA基础资源指标的配置，通过指定ResourceName表示以哪种资源为指标，ResourceName支持cpu和memory，TargetType支持Utilization和AverageValue
* CustomMetrics是以自定义metric为指标，通过指定MetricName表示以哪一个自定义metric为指标，只支持AverageValue
* 无论HPA如何动态调整workload的pods个数，所调整的pods个数都不会超过MinReplicas和MaxReplicas的范围。

## 配置范例

	"name": "hpa-vg1-vanguard",
    "scaleTargetKind": "deployment",
    "scaleTargetName": "vg1-vanguard",
    "minReplicas": 1,
    "maxReplicas": 5,
    "customMetrics":[
    	{
    		"metricName": "zdns_vanguard_qps",
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