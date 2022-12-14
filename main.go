package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/henderiw/fabric/fabric"
	"github.com/yndd/ndd-runtime/pkg/logging"
	topov1alpha1 "github.com/yndd/topology/apis/topo/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	zlog := zap.New(zap.UseDevMode(true), zap.JSONEncoder())
	logger := logging.NewLogrLogger(zlog.WithName("fabric"))

	d, err := os.ReadFile("./example/template.json")
	if err != nil {
		panic(err)
	}
	//fmt.Printf("raw ytes: \n%s\n", string(d))

	t := topov1alpha1.Template{}
	if err := json.Unmarshal(d, &t); err != nil {
		panic(err)
	}

	f, err := fabric.New(&t,
		fabric.WithLogger(logger),
		fabric.WithLocation(&topov1alpha1.Location{
			Latitude:  "51.090875423265956",
			Longitude: "4.87314214079595",
		}),
	)
	if err != nil {
		panic(err)
	}
	f.PrintNodes()
	f.PrintLinks()

	fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@")
	fmt.Println()
	f.PrintGraph()

	if err := f.GenerateJsonFile(); err != nil {
		panic(err)
	}
}
