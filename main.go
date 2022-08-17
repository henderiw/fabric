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

	t := topov1alpha1.FabricTemplate{}
	if err := json.Unmarshal(d, &t); err != nil {
		panic(err)
	}

	f, err := fabric.New("nokia.region1.fabric1", &t,
		fabric.WithLogger(logger),
	)
	if err != nil {
		panic(err)
	}
	f.PrintNodes()
	f.PrintLinks()

	fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@")
	fmt.Println()
	f.PrintGraph()
}
