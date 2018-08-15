package operator

// some doc here https://thenewstack.io/extend-kubernetes-1-7-custom-resources/

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/websocket"

	apiuseless "github.com/vasu1124/introspect/pkg/operator/apis/useless"
	uselessv1 "github.com/vasu1124/introspect/pkg/operator/apis/useless/v1"
	"github.com/vasu1124/introspect/pkg/version"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Handler .
type Handler struct {
	controller *UselessController
}

// New .
func New() *Handler {
	var h Handler
	var err error

	// Create the client config. Use masterURL and kubeconfig if given, otherwise assume in-cluster.
	config, err := clientcmd.BuildConfigFromFlags(*version.MasterURL, *version.Kubeconfig)
	if err != nil {
		log.Printf("[operator] KubeConfig error: %v", err)
		return &h
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		log.Printf("[operator] ClientSet error: %v", err)
		return &h
	}

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	crd, err := CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		log.Printf("[operator] CRD create error: %v", err)
		return &h
	}
	if crd != nil {
		log.Printf("[operator] created CRD: %v", crd)
	}

	/* 	if crd != nil {
	   		defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, nil)
	   	}
	*/

	scheme := runtime.NewScheme()
	if err := uselessv1.AddToScheme(scheme); err != nil {
		log.Printf("[operator] Scheme error: %v", err)
		return &h
	}

	config.GroupVersion = &uselessv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		log.Printf("[operator] REST client error: %v", err)
		return &h
	}

	// start a UselessController on instances of our custom resource
	h.controller = &UselessController{
		uselessClient: client,
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	//	defer cancelFunc()
	go h.controller.Run(ctx)
	log.Printf("[operator] not used canceFunc: %v", cancelFunc)

	return &h
}

const uselessCRDName = uselessv1.UselessResourcePlural + "." + apiuseless.GroupName

//CreateCustomResourceDefinition .
func CreateCustomResourceDefinition(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: uselessCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   uselessv1.SchemeGroupVersion.Group,
			Version: uselessv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: uselessv1.UselessResourcePlural,
				Kind:   reflect.TypeOf(uselessv1.Useless{}).Name(),
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(uselessCRDName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					log.Printf("[operator] CRD Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(uselessCRDName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

//ServeHTTP .
//This is an UI to the state of the Operator
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil {
		fmt.Fprint(w, "[operator] No active Controller")
		return
	}

	_, err := h.controller.List()
	if err != nil {
		fmt.Fprint(w, "[operator] No working Controller")
		return
	}

	type EnvData struct {
		version string
	}

	data := EnvData{version.Version}

	homeTemplate, err := template.ParseFiles("tmpl/operator.html")
	if err != nil {
		fmt.Fprint(w, "[operator] parsing template: ", err)
		log.Println("[operator] parsing template: ", err)
		return
	}

	err = homeTemplate.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, "[operator] executing template: ", err)
		log.Println("[operator] executing template: ", err)
	}

}

var upgrader = websocket.Upgrader{} // use default options

//ServeWS .
//Here we handle synchronous requests from or UI
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	var err error
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[operator] websocket upgrade:", err)
		return
	}
	log.Println("[operator] websocket connected")
	h.controller.RegisterUI(conn)

	stop := make(chan struct{})
	in := make(chan uselessv1.UselessList)

	ticker := time.NewTicker(40 * time.Second)

	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	go func() {
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[operator] websocket read err: %v %v", mt, err)
				close(stop)
				break
			}
			// log.Printf("[operator] websocket recv: %s", message)

			var msg uselessv1.UselessList
			err = json.Unmarshal(message, &msg)
			if err != nil {
				log.Printf("[operator] websocket unmarshall: %v", err)
				break
			}
			in <- msg
		}
		log.Println("[operator] stop reading of connection from", conn.RemoteAddr())
	}()

	for {
		select {
		case msg := <-in:
			switch msg.Kind {
			case "GET_ITEMS":
				list, _ := h.controller.List()
				listmarshall, _ := json.Marshal(list)
				err = conn.WriteMessage(websocket.TextMessage, listmarshall)
				if err != nil {
					log.Println("[operator] websocket write: ", err)
					break
				}
			case "CHANGED_ITEMS":
				list, _ := h.controller.List()
				//changes from UI /citem
				for _, citem := range msg.Items {
					//compare with original backend /oitem
					for _, oitem := range list.Items {
						//find the match (TODO: could be optimized with a set/lookup)
						if oitem.ObjectMeta.UID == citem.ObjectMeta.UID {
							if oitem.Status.State != citem.Status.State {
								//found the changed item in the original list
								log.Println("[operator] changed UID: ", citem.ObjectMeta.UID)

								//change the backend item to state changed by UI
								h.controller.SetState(&oitem, &uselessv1.UselessStatus{
									Message: "Change processed by UI",
									State:   citem.Status.State,
								})
							}
							break
						}
					}
				}
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		case <-stop:
			break
		}
	}
}
