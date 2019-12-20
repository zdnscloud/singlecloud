# workload metrics
## 概要
workload可以暴露自己的metrics，通过p配置pod暴露出来的metric port和path，就可以从pod获取出workload所有的metrics，由于需要通过pod的ip和port获取metrics，所以singlecloud通过zcloud proxy向cluster agent请求workload的metrics

## 详细设计
### Metric资源
* metric为workload下的子资源，即deployment、daemonset、statefulset的子资源
* metric属性

		type Metric struct {
    		Name    string         `json:"name,omitempty"`
    		Type    string         `json:"type,omitempty"`
    		Help    string         `json:"help,omitempty"`
    		Metrics []MetricFamily `json:"metrics,omitempty"`
		}

		type MetricFamily struct {
    		Labels  map[string]string `json:"labels,omitempty"`
    		Gauge   Gauge             `json:"gauge,omitempty"`
    		Counter Counter           `json:"counter,omitempty"`
		}

		type Gauge struct {
    		Value int `json:"value,omitempty"`
		}

		type Counter struct {
    		Value int `json:"value,omitempty"`
		}

* workload要想暴露metric，需要在workload的Spec.Annotations中配置如下配置
  * prometheus.io/path: 暴露metric的path，默认值为metrics
  * prometheus.io/port: 暴露metric的port
  
		spec:
		  template:
    		metadata:
			  annotations:
        	    prometheus.io/port: "9000"
        		prometheus.io/path: "/metrics"
