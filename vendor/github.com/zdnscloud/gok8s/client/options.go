package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

type DeleteOptions struct {
	// The value zero indicates delete immediately.
	GracePeriodSeconds *int64

	// Preconditions must be fulfilled before a deletion is carried out. If not
	// possible, a 409 Conflict status will be returned.
	Preconditions *metav1.Preconditions

	// 'Orphan' - orphan the dependents;
	// 'Background' - allow the garbage collector to delete the dependents in the background;
	// 'Foreground' - a cascading policy that deletes all dependents in the foreground.
	PropagationPolicy *metav1.DeletionPropagation

	// Raw represents raw DeleteOptions, as passed to the API server.
	Raw *metav1.DeleteOptions
}

func (o *DeleteOptions) AsDeleteOptions() *metav1.DeleteOptions {
	if o == nil {
		return &metav1.DeleteOptions{}
	}

	if o.Raw == nil {
		o.Raw = &metav1.DeleteOptions{}
	}

	o.Raw.GracePeriodSeconds = o.GracePeriodSeconds
	o.Raw.Preconditions = o.Preconditions
	o.Raw.PropagationPolicy = o.PropagationPolicy
	return o.Raw
}

func (o *DeleteOptions) ApplyOptions(optFuncs []DeleteOptionFunc) *DeleteOptions {
	for _, optFunc := range optFuncs {
		optFunc(o)
	}
	return o
}

type DeleteOptionFunc func(*DeleteOptions)

func GracePeriodSeconds(gp int64) DeleteOptionFunc {
	return func(opts *DeleteOptions) {
		opts.GracePeriodSeconds = &gp
	}
}

func Preconditions(p *metav1.Preconditions) DeleteOptionFunc {
	return func(opts *DeleteOptions) {
		opts.Preconditions = p
	}
}

func PropagationPolicy(p metav1.DeletionPropagation) DeleteOptionFunc {
	return func(opts *DeleteOptions) {
		opts.PropagationPolicy = &p
	}
}

type ListOptions struct {
	LabelSelector labels.Selector
	FieldSelector fields.Selector
	Namespace     string
	Raw           *metav1.ListOptions
}

func (o *ListOptions) SetLabelSelector(selRaw string) error {
	sel, err := labels.Parse(selRaw)
	if err != nil {
		return err
	}
	o.LabelSelector = sel
	return nil
}

func (o *ListOptions) SetFieldSelector(selRaw string) error {
	sel, err := fields.ParseSelector(selRaw)
	if err != nil {
		return err
	}
	o.FieldSelector = sel
	return nil
}

func (o *ListOptions) AsListOptions() *metav1.ListOptions {
	if o == nil {
		return &metav1.ListOptions{}
	}
	if o.Raw == nil {
		o.Raw = &metav1.ListOptions{}
	}
	if o.LabelSelector != nil {
		o.Raw.LabelSelector = o.LabelSelector.String()
	}
	if o.FieldSelector != nil {
		o.Raw.FieldSelector = o.FieldSelector.String()
	}
	return o.Raw
}

func (o *ListOptions) MatchingLabels(lbls map[string]string) *ListOptions {
	sel := labels.SelectorFromSet(lbls)
	o.LabelSelector = sel
	return o
}

func (o *ListOptions) MatchingField(name, val string) *ListOptions {
	sel := fields.SelectorFromSet(fields.Set{name: val})
	o.FieldSelector = sel
	return o
}

func (o *ListOptions) InNamespace(ns string) *ListOptions {
	o.Namespace = ns
	return o
}

func MatchingLabels(lbls map[string]string) *ListOptions {
	return (&ListOptions{}).MatchingLabels(lbls)
}

func MatchingField(name, val string) *ListOptions {
	return (&ListOptions{}).MatchingField(name, val)
}

func InNamespace(ns string) *ListOptions {
	return (&ListOptions{}).InNamespace(ns)
}
