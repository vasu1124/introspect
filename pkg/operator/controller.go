package operator

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/vasu1124/introspect/pkg/election"
	uselessv1 "github.com/vasu1124/introspect/pkg/operator/apis/useless/v1"
)

// UselessController Watcher is an UselessController of watching on resource create/update/delete events
type UselessController struct {
	uselessClient *rest.RESTClient
	uiconns       []*websocket.Conn
}

// Run starts an Useless resource controller
func (c *UselessController) Run(ctx context.Context) error {
	log.Print("[controller] Watch Useless objects\n")

	// Watch Useless objects
	_, err := c.watchUselesss(ctx)
	if err != nil {
		log.Printf("[controller] Failed to register watch for Useless resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *UselessController) watchUselesss(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.uselessClient,
		uselessv1.UselessResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&uselessv1.Useless{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *UselessController) onAdd(obj interface{}) {
	useless := obj.(*uselessv1.Useless)
	log.Printf("[controller] OnAdd %s\n", useless.ObjectMeta.SelfLink)
	if election.Leader && useless.Status.StateCode == "" {
		// encountering a fresh add, which needs to be initialized

		c.SetState(useless, &uselessv1.UselessStatus{
			Message: "Successfully processed by introspect controller",
			State:   useless.Spec.DesiredState,
		})

		c.sendUIUpdate()
	}
}

func (c *UselessController) onUpdate(oldObj, newObj interface{}) {
	oldUseless := oldObj.(*uselessv1.Useless)
	newUseless := newObj.(*uselessv1.Useless)

	if oldUseless.Status.State != newUseless.Status.State ||
		oldUseless.Spec.DesiredState != newUseless.Spec.DesiredState {
		log.Printf("[controller] OnUpdate %s, old:%v new:%v\n", oldUseless.ObjectMeta.SelfLink, oldUseless.Status, newUseless.Status)

		c.sendUIUpdate()

		if election.Leader {
			//correct the situation! BUt don't do it quick ;-)
			time.Sleep(2000 * time.Millisecond)
			c.SetState(newUseless, &uselessv1.UselessStatus{
				Message: "Successfully corrected by introspect controller",
				State:   newUseless.Spec.DesiredState,
			})
		}
	}
}

func (c *UselessController) onDelete(obj interface{}) {
	useless := obj.(*uselessv1.Useless)
	log.Printf("[controller] OnDelete %s\n", useless.ObjectMeta.SelfLink)

	c.sendUIUpdate()
}

//Get .
func (c *UselessController) Get(name string) (*uselessv1.Useless, error) {
	var result uselessv1.Useless
	err := c.uselessClient.Get().
		Resource(uselessv1.UselessResourcePlural).
		Name(name).Do().Into(&result)
	return &result, err
}

//List .
func (c *UselessController) List() (*uselessv1.UselessList, error) {
	var result uselessv1.UselessList
	err := c.uselessClient.Get().
		Resource(uselessv1.UselessResourcePlural).
		Do().Into(&result)
	result.Kind = "UselessList"
	return &result, err
}

//SetState .
func (c *UselessController) SetState(useless *uselessv1.Useless, stat *uselessv1.UselessStatus) {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	uselessCopy := useless.DeepCopy()
	if stat.State == uselessCopy.Spec.DesiredState {
		stat.StateCode = uselessv1.UselessStateOK
	} else {
		stat.StateCode = uselessv1.UselessStateNOK
	}
	uselessCopy.Status = *stat

	err := c.uselessClient.Put().
		Name(useless.ObjectMeta.Name).
		Namespace(useless.ObjectMeta.Namespace).
		Resource(uselessv1.UselessResourcePlural).
		Body(uselessCopy).
		Do().
		Error()

	if err != nil {
		log.Printf("[controller] ERROR updating status: %v\n", err)
	} else {
		log.Printf("[controller] UPDATED status: %#v\n", uselessCopy.Status)
	}
}

// UI related

//RegisterUI .
//TODO: secure with mutex
func (c *UselessController) RegisterUI(conn *websocket.Conn) {
	c.uiconns = append(c.uiconns, conn)
}

//DeregisterUI .
//TODO: secure with mutex
func (c *UselessController) DeregisterUI(i int) {
	c.uiconns[i].Close()
	c.uiconns = append(c.uiconns[:i], c.uiconns[i+1:]...)
}

func (c *UselessController) sendUIUpdate() {
	if c.uiconns == nil {
		return
	}

	list, _ := c.List()
	x, _ := json.Marshal(list)

	for i, uiconn := range c.uiconns {
		err := uiconn.WriteMessage(websocket.TextMessage, x)
		if err != nil {
			//log.Println("[controller] websocket write:", err)
			//remove errored ws ui connection from array
			c.DeregisterUI(i)
		}
	}
}
