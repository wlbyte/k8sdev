package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ingress-by-service/pkg"

	apisv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. 创建config
	// 2. 创建clientset
	// 3. 创建informer
	// 4. 给informer添加处理器
	// 5. 启动informer
	clientset := newClientSet()
	// factory := informers.NewSharedInformerFactory(clientset, time.Minute)
	factory := informers.NewFilteredSharedInformerFactory(clientset, time.Minute, apisv1.NamespaceDefault, nil)
	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientset, serviceInformer, ingressInformer)
	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGALRM)
	factory.Start(stopChan)
	factory.WaitForCacheSync(stopChan)

	log.Println("custom controller start...")
	controller.Run(stopChan, sigChan)
}

func newClientSet() *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal("new client error: ", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("new clientset error: ", err)
	}
	return clientset
}
