package main

import (
	"context"
	"log"

	apisV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. 创建config
	// 2. 创建clientset
	clientset := newClientSet()
	podsList, err := clientset.CoreV1().Pods("").List(context.TODO(), apisV1.ListOptions{})
	if err != nil {
		log.Fatal("get pods error:", err.Error())
	}
	for _, pod := range podsList.Items {
		log.Printf("%s", pod.Name)
	}
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
