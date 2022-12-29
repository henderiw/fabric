package main

import (
	"fmt"
	"os"

	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
	"github.com/henderiw/fabric/fabric"
	"sigs.k8s.io/yaml"
)

func main() {
	//zlog := zap.New(zap.UseDevMode(true), zap.JSONEncoder())
	//logger := logging.NewLogrLogger(zlog.WithName("fabric"))

	d, err := os.ReadFile("./example/template.yaml")
	if err != nil {
		panic(err)
	}
	//fmt.Printf("raw ytes: \n%s\n", string(d))

	t := &topov1alpha1.Template{}
	if err := yaml.Unmarshal(d, t); err != nil {
		panic(err)
	}

	f, err := fabric.New(&fabric.Config{
		Name: "fabric1",
		Namespace: "default",
		ChildTemplates: []*topov1alpha1.Template{
			t,
		},
		Location: &topov1alpha1.Location{
			Latitude:  "a",
			Longitude: "b",
		},
	})
	if err != nil {
		panic(err)
	}
	//f.PrintNodes()
	//f.PrintLinks()

	fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@")
	fmt.Println()
	f.PrintGraph()

	if err := f.GenerateJsonFile(); err != nil {
		panic(err)
	}

	f.Print()
}
