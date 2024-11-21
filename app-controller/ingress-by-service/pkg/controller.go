package pkg

import (
	"context"
	"log"
	"os"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerCoreV1 "k8s.io/client-go/informers/core/v1"
	informerNetV1 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	listerNetV1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// worker 数量
	workNum = 5
	// service 指定 ingress 的 annotation key
	annoKey = "ingress/http"
	// 调谐失败的最大重试次数
	maxRetry = 10
)

type controller struct {
	client        kubernetes.Interface
	serviceLister listerCoreV1.ServiceLister
	ingressLister listerNetV1.IngressLister
	queue         workqueue.RateLimitingInterface
}

func NewController(client kubernetes.Interface, serviceInformer informerCoreV1.ServiceInformer, ingressInformer informerNetV1.IngressInformer) *controller {
	c := &controller{
		client:        client,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}

func (c *controller) addService(obj any) {
	svc := obj.(*corev1.Service)
	log.Printf("addService obj: %v\n", svc.ObjectMeta.Name)
	c.enqueue(obj)
}

func (c *controller) updateService(oldObj any, newObj any) {
	if reflect.DeepEqual(oldObj, newObj) {
		log.Println("updateService obj is same, ignore")
		return
	}
	c.enqueue(newObj)
}

func (c *controller) deleteIngress(obj any) {
	ingress := obj.(*netv1.Ingress)
	log.Printf("deleteIngress obj: %v\n", ingress.ObjectMeta.Name)
	ownerReference := metav1.GetControllerOf(ingress)
	if ownerReference == nil || ownerReference.Kind != "Service" {
		log.Println("deleteIngress, ignore delete")
		return
	}
	log.Println("deleteIngress to queue")
	c.enqueue(obj)

}

func (c *controller) enqueue(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *controller) dequeue(obj any) {
	c.queue.Done(obj)
}

func (c *controller) Run(stopChan chan struct{}, sigChan chan os.Signal) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopChan)
	}
	select {
	case <-stopChan:
		return
	case sig := <-sigChan:
		log.Println("recieve signal ", sig, ", controller exit")
		return
	}
}

func (c *controller) worker() {
	for c.processNextItem() {

	}
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.dequeue(item)
	key := item.(string)
	log.Println("process item: ", key)
	err := c.syncService(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *controller) handleError(key string, err error) {
	if c.queue.NumRequeues(key) < maxRetry {
		c.queue.AddRateLimited(key)
		return
	}
	runtime.HandleError(err)
	c.queue.Forget(key)
}

func (c *controller) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	log.Printf("item -> namespace: %s, name: %s\n", namespace, name)
	service, err := c.serviceLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, ok := service.Annotations[annoKey]
	ingress, err := c.ingressLister.Ingresses(namespace).Get(name)
	if ok && errors.IsNotFound(err) {
		log.Printf("annoExist: ok, ingress: not found\n")
		ig := c.createIngress(service)
		log.Println("createIngress", ig.ObjectMeta.Name)
		_, err := c.client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ig, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		log.Println("deleteIngress", ingress.ObjectMeta.Name)
		err := c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// createIngress 创建ingress
func (c *controller) createIngress(service *corev1.Service) *netv1.Ingress {
	icn := "nginx"
	pathType := netv1.PathTypePrefix
	return &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service")),
			},
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &icn,
			Rules: []netv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: service.Name,
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
