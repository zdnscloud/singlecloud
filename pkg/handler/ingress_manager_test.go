package handler

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestIngressRuleValidate(t *testing.T) {
	cases := []struct {
		ing          types.Ingress
		isValid      bool
		k8sRuleCount int
	}{
		{types.Ingress{
			Name: "i1",
			Rules: []types.IngressRule{
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, true, 1},

		{types.Ingress{
			Name: "i1",
			Rules: []types.IngressRule{
				types.IngressRule{
					Host:        "xx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, true, 2},

		{types.Ingress{
			Name: "i1",
			Rules: []types.IngressRule{
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, false, 1},

		{types.Ingress{
			Name: "i1",
			Rules: []types.IngressRule{
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xx",
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, true, 2},
		{types.Ingress{
			Name: "i1",
			Rules: []types.IngressRule{
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xx",
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressRule{
					Host:        "xxx",
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, false, 2},
	}

	for _, tc := range cases {
		ing, err := scIngressTok8sIngress("n1", &tc.ing)
		if !tc.isValid {
			ut.Assert(t, err != nil, "should err for case %s, but get nothing", tc.ing)
		} else {
			ut.Assert(t, err == nil, "should ok but get %v", err)
			ut.Equal(t, len(ing.Spec.Rules), tc.k8sRuleCount)
		}
	}
}
