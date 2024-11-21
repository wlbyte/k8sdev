package main

import (
	clientset "code-generator-crd/pkg/generated/clientset/versioned"
	"context"
	"log"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}
	list, err := clientset.CrdV1().Foos("default").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, foo := range list.Items {
		println(foo.Name)
	}

	// todo 注册informer

}
