package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1" // 导入v1包
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func main() {
	// kubeconfig认证方式
	// kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "absolute path to the kubeconfig file")
	// masterURL := flag.String("master", "", "The address of the Kubernetes API server")
	// flag.Parse()
	// if _, err := os.Stat(*kubeconfig); err != nil {
	// 	log.Fatalf("[Error] k8s %s is not exist", *kubeconfig)
	// }
	// config, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	// if err != nil {
	// 	log.Fatalf("building kubeconfig error: %s", err)
	// }

	// 集群内认证
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("InClusterConfig() error: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err)
	}

	// 查询endpoints资源
	// endpoints, err := clientset.CoreV1().Endpoints("").List(context.TODO(), v1.ListOptions{})
	// if err != nil {
	// 	log.Fatalf("clientset.CoreV1().Endpoints() error: %s", err)
	// }
	// for _, ep := range endpoints.Items {
	// 	if len(ep.Subsets) < 1 || len(ep.Subsets[0].Ports) < 1 {
	// 		continue
	// 	}
	// 	port := ep.Subsets[0].Ports[0].Port
	// 	protocol := ep.Subsets[0].Ports[0].Protocol
	// 	fmt.Printf("name: %s, namesapce: %s, protocol: %s endpoint: ", ep.ObjectMeta.Name, ep.ObjectMeta.Namespace, protocol)
	// 	for _, ip := range ep.Subsets[0].Addresses {
	// 		fmt.Printf("%s:%d,", ip.IP, port)
	// 	}
	// 	fmt.Println()
	// }

	//查询pods
	// services, err := clientset.CoreV1().Services("").List(context.TODO(), v1.ListOptions{})
	// if err != nil {
	// 	log.Fatalf("clientset.CoreV1().Pods().List() error: %s", err)
	// }
	// for _, service := range services.Items {
	// 	fmt.Println(service)
	// }

	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)
	endpointsInformer := informerFactory.Core().V1().Endpoints().Informer()

	endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    reportEndpointsAddFunc,
		UpdateFunc: reportEndpointsUpdateFunc,
		DeleteFunc: reportEndpointsDelFunc,
	})

	stopChan := make(chan struct{})
	defer close(stopChan)
	informerFactory.Start(stopChan)
	// 等待数据同步完成
	if !cache.WaitForCacheSync(stopChan, endpointsInformer.HasSynced) {
		fmt.Println("sync failed")
		return
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	<-signalChan

}

func reportEndpointsAddFunc(obj interface{}) {
	endpoints := obj.(*v1.Endpoints)
	fmt.Printf("endpoints added: %s/%s\n", endpoints.Namespace, endpoints.Name)
	printEndpointInfo(endpoints)
}

func reportEndpointsUpdateFunc(oldObj, newObj interface{}) {
	endpointsOld := oldObj.(*v1.Endpoints)
	endpointsNew := newObj.(*v1.Endpoints)
	if strings.TrimSpace(getEndpointAddr(endpointsOld)) != strings.TrimSpace(getEndpointAddr(endpointsNew)) {
		fmt.Printf("endpoints update: %s/%s\n", endpointsOld.Namespace, endpointsOld.Name)
		printEndpointInfo(endpointsOld)
		printEndpointInfo(endpointsNew)
	}
}

func reportEndpointsDelFunc(obj interface{}) {
	endpoints := obj.(*v1.Endpoints)
	fmt.Printf("endpoint del: %s/%s\n", endpoints.Namespace, endpoints.Name)
	printEndpointInfo(endpoints)
}

// 获取endpoints资源相关信息
func printEndpointInfo(ep *v1.Endpoints) {
	fmt.Println(getEndpointAddr(ep))

}

func getEndpointAddr(ep *v1.Endpoints) string {
	if len(ep.Subsets) < 1 || len(ep.Subsets[0].Ports) < 1 {
		return ""
	}
	var addr string
	port := ep.Subsets[0].Ports[0].Port
	addr = string(ep.Subsets[0].Ports[0].Protocol) + "/"
	for _, ip := range ep.Subsets[0].Addresses {
		addr += fmt.Sprintf("%s:%s,", ip.IP, strconv.Itoa(int(port)))
	}
	return strings.TrimRight(addr, ",")
}
