package handler

import (
	"sort"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestIngressRuleValidate(t *testing.T) {
	cases := []struct {
		ing         types.IngressRule
		shouldError bool
	}{
		{types.IngressRule{
			Host:     "dd",
			Port:     10,
			Protocol: types.IngressProtocolUDP,
		}, true},

		{types.IngressRule{
			Port:     0,
			Protocol: types.IngressProtocolUDP,
		}, true},

		{types.IngressRule{
			Host:     "xxx",
			Protocol: types.IngressProtocolUDP,
		}, true},

		{types.IngressRule{
			Port:     20,
			Protocol: types.IngressProtocolHTTP,
		}, true},

		{types.IngressRule{
			Host:     "xxx",
			Protocol: types.IngressProtocolHTTP,
		}, true},

		{types.IngressRule{
			Host:     "xxx",
			Protocol: types.IngressProtocolHTTP,
			Paths: []types.IngressPath{
				types.IngressPath{
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressPath{
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, true},

		{types.IngressRule{
			Port:     10,
			Protocol: types.IngressProtocolUDP,
			Paths: []types.IngressPath{
				types.IngressPath{
					ServiceName: "xxx",
					ServicePort: 30,
				},
			},
		}, false},

		{types.IngressRule{
			Host:     "xxx",
			Protocol: types.IngressProtocolHTTP,
			Paths: []types.IngressPath{
				types.IngressPath{
					Path:        "/v2",
					ServiceName: "xxx",
					ServicePort: 30,
				},
				types.IngressPath{
					Path:        "/v1",
					ServiceName: "xxx",
					ServicePort: 20,
				},
			},
		}, false},
	}

	for _, tc := range cases {
		err := validateRule(&tc.ing)
		if tc.shouldError {
			ut.Assert(t, err != nil, "should err for case %s, but get nothing", tc.ing)
		} else {
			ut.Assert(t, err == nil, "should ok but get %v", err)
		}
	}
}

type ByPath []types.IngressPath

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }

type ByProtocolHostPort []types.IngressRule

func (a ByProtocolHostPort) Len() int      { return len(a) }
func (a ByProtocolHostPort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByProtocolHostPort) Less(i, j int) bool {
	ra := a[i]
	rb := a[j]
	if ra.Protocol != rb.Protocol {
		return ra.Protocol < rb.Protocol
	}
	if ra.Host != rb.Host {
		return ra.Host < rb.Host
	}
	return ra.Port < rb.Port
}

func ingressRuleEqual(ra, rb *types.IngressRule) bool {
	if ra.Protocol != rb.Protocol || ra.Host != rb.Host || ra.Port != rb.Port {
		return false
	}

	if len(ra.Paths) != len(rb.Paths) {
		return false
	}

	aPaths := make([]types.IngressPath, len(ra.Paths))
	copy(aPaths, ra.Paths)
	bPaths := make([]types.IngressPath, len(rb.Paths))
	copy(bPaths, rb.Paths)
	sort.Sort(ByPath(aPaths))
	sort.Sort(ByPath(bPaths))
	for i, p := range aPaths {
		if p != bPaths[i] {
			return false
		}
	}
	return true
}

func TestIngressRuleMerge(t *testing.T) {
	cases := []struct {
		ruleA       types.IngressRule
		ruleB       types.IngressRule
		result      *types.IngressRule
		merged      bool
		shouldError bool
	}{
		{
			types.IngressRule{
				Protocol: types.IngressProtocolUDP,
				Port:     10,
			},
			types.IngressRule{
				Protocol: types.IngressProtocolUDP,
				Port:     10,
			},
			nil,
			false,
			true,
		},
		{
			types.IngressRule{
				Protocol: types.IngressProtocolUDP,
				Port:     10,
			},
			types.IngressRule{
				Protocol: types.IngressProtocolTCP,
				Port:     10,
			},
			nil,
			false,
			false,
		},
		{
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
			},
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a2",
			},
			nil,
			false,
			false,
		},

		{
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        "/v1",
						ServiceName: "xxx",
						ServicePort: 22,
					},
				},
			},
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        "/v2",
						ServiceName: "xxx1",
						ServicePort: 30,
					},
					types.IngressPath{
						Path:        "/v1",
						ServiceName: "xxx2",
						ServicePort: 30,
					},
				},
			},
			nil,
			false,
			true,
		},
		{
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        "/v1",
						ServiceName: "xxx",
						ServicePort: 22,
					},
				},
			},
			types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        "/v2",
						ServiceName: "xxx1",
						ServicePort: 30,
					},
					types.IngressPath{
						Path:        "/v3",
						ServiceName: "xxx2",
						ServicePort: 30,
					},
				},
			},
			&types.IngressRule{
				Protocol: types.IngressProtocolHTTP,
				Host:     "a1",
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        "/v1",
						ServiceName: "xxx",
						ServicePort: 22,
					},
					types.IngressPath{
						Path:        "/v2",
						ServiceName: "xxx1",
						ServicePort: 30,
					},
					types.IngressPath{
						Path:        "/v3",
						ServiceName: "xxx2",
						ServicePort: 30,
					},
				},
			},
			true,
			false,
		},
	}

	for _, tc := range cases {
		merged, err := mergeIngressRule(&tc.ruleA, &tc.ruleB)
		ut.Equal(t, merged, tc.merged)

		if tc.merged {
			ut.Assert(t, ingressRuleEqual(tc.result, &tc.ruleB), "")
		}

		if tc.shouldError {
			ut.Assert(t, err != nil, "should err for case %v : %v, but get nothing", tc.ruleA, tc.ruleB)
		} else {
			ut.Assert(t, err == nil, "should ok but get %v", err)
		}
	}
}

func TestMergeIngressRules(t *testing.T) {
	cases := []struct {
		before      []types.IngressRule
		after       []types.IngressRule
		shouldError bool
	}{
		{nil, nil, false},
		{
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v2",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
						types.IngressPath{
							Path:        "/v2",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			false,
		},
		{
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolUDP,
					Port:     30,
					Paths: []types.IngressPath{
						types.IngressPath{
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolUDP,
					Port:     30,
					Paths: []types.IngressPath{
						types.IngressPath{
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			false,
		},
		{
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a2",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolUDP,
					Port:     30,
					Paths: []types.IngressPath{
						types.IngressPath{
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v2",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolUDP,
					Port:     30,
					Paths: []types.IngressPath{
						types.IngressPath{
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
						types.IngressPath{
							Path:        "/v2",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a2",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			false,
		},

		{
			[]types.IngressRule{
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a1",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a2",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolUDP,
					Port:     30,
					Paths: []types.IngressPath{
						types.IngressPath{
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
				types.IngressRule{
					Protocol: types.IngressProtocolHTTP,
					Host:     "a2",
					Paths: []types.IngressPath{
						types.IngressPath{
							Path:        "/v1",
							ServiceName: "xxx",
							ServicePort: 22,
						},
					},
				},
			},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		merged, err := mergeIngressRules(tc.before)
		if tc.shouldError {
			ut.Assert(t, err != nil, "should err for case %v but get nothing", tc.before)
		} else {
			ut.Assert(t, err == nil, "should ok but get %v", err)
			sort.Sort(ByProtocolHostPort(merged))
			sort.Sort(ByProtocolHostPort(tc.after))
			ut.Equal(t, len(merged), len(tc.after))
			for i, _ := range merged {
				ut.Assert(t, ingressRuleEqual(&merged[i], &tc.after[i]), "")
			}
		}
	}
}
