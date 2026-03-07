package main

import (
	"fmt"

	"github.com/kubenas/kubenas/operator/controllers"
)

func main() {
	reconcilers := []controllers.Reconciler{
		&controllers.DiskController{},
		&controllers.ArrayController{},
		&controllers.PoolController{},
		&controllers.ShareController{},
		&controllers.ParityController{},
	}

	fmt.Println("KubeNAS operator bootstrap initialized with controllers:")
	for _, r := range reconcilers {
		fmt.Printf("- %s\n", r.Name())
	}
}
