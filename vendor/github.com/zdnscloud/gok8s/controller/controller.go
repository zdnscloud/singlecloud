package controller

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/apiutil"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/eventsource"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
)

type controller struct {
	name       string
	handler    handler.EventHandler
	predicates []predicate.Predicate
	cache      cache.Cache
	sources    map[schema.GroupVersionKind]<-chan interface{}
	queue      workqueue.RateLimitingInterface
	scheme     *runtime.Scheme
}

func New(name string, cache cache.Cache, scheme *runtime.Scheme) Controller {
	return &controller{
		name:    name,
		cache:   cache,
		sources: make(map[schema.GroupVersionKind]<-chan interface{}),
		scheme:  scheme,
	}
}

func (c *controller) Watch(obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	if _, ok := c.sources[gvk]; ok {
		return fmt.Errorf("watch obj %v more than once", gvk)
	}

	ch, err := eventsource.New(gvk, c.cache).GetEventChannel()
	if err != nil {
		return err
	}

	c.sources[gvk] = ch
	return nil
}

func (c *controller) Start(stop <-chan struct{}, handler handler.EventHandler, predicates ...predicate.Predicate) {
	c.handler = handler
	c.predicates = predicates
	c.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), c.name)

	var wg wait.Group
	wg.StartWithChannel(stop, c.collectEvent)
	wg.StartWithChannel(stop, c.processEvent)
	wg.Wait()
}

func (c *controller) collectEvent(stop <-chan struct{}) {
	cases := make([]reflect.SelectCase, 0, len(c.sources)+1)
	for _, ch := range c.sources {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(stop),
	})

	for len(cases) > 0 {
		i, e, ok := reflect.Select(cases)
		if i == len(cases)-1 {
			c.queue.ShutDown()
			return
		}

		if !ok {
			cases = append(cases[:i], cases[i+1:]...)
			continue
		}

		c.queue.Add(e.Interface())
	}
}

func (c *controller) processEvent(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		c.processNextEvent()
	}
}

func (c *controller) processNextEvent() {
	o, shutdown := c.queue.Get()
	if shutdown {
		return
	}
	defer c.queue.Done(o)

	if o == nil {
		c.queue.Forget(o)
		return
	}

	var err error
	var result handler.Result
	switch e := o.(type) {
	case event.CreateEvent:
		for _, p := range c.predicates {
			if p.IgnoreCreate(e) {
				return
			}
		}
		result, err = c.handler.OnCreate(e)
	case event.UpdateEvent:
		for _, p := range c.predicates {
			if p.IgnoreUpdate(e) {
				return
			}
		}
		result, err = c.handler.OnUpdate(e)
	case event.DeleteEvent:
		for _, p := range c.predicates {
			if p.IgnoreDelete(e) {
				return
			}
		}
		result, err = c.handler.OnDelete(e)
	case event.GenericEvent:
		result, err = c.handler.OnGeneric(e)

	default:
		panic(fmt.Sprintf("unkown event [%v]", reflect.TypeOf(o).Name()))
	}

	if err != nil {
		c.queue.AddRateLimited(o)
	} else if result.RequeueAfter > 0 {
		c.queue.AddAfter(o, result.RequeueAfter)
	} else if result.Requeue {
		c.queue.AddRateLimited(o)
	} else {
		c.queue.Forget(o)
	}
}
