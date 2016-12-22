package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Mirantis/k8s-AppController/client"
	"github.com/Mirantis/k8s-AppController/scheduler"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/labels"
)

// GetStatus is a command that prints the deployment status
func getStatus(cmd *cobra.Command, args []string) {
	var err error

	labelSelector, err := getLabelSelector(cmd)
	if err != nil {
		log.Fatal(err)
	}

	getJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		log.Fatal(err)
	}
	getReport, err := cmd.Flags().GetBool("report")
	if err != nil {
		log.Fatal(err)
	}

	var url string
	if len(args) > 0 {
		url = args[0]
	}
	if url == "" {
		url = os.Getenv("KUBERNETES_CLUSTER_URL")
	}

	c, err := client.New(url)
	if err != nil {
		log.Fatal(err)
	}
	sel, err := labels.Parse(labelSelector)
	if err != nil {
		log.Fatal(err)
	}
	graph, err := scheduler.BuildDependencyGraph(c, sel)
	if err != nil {
		log.Fatal(err)
	}
	status, report := graph.GetStatus()
	if getJSON {
		data, err := json.Marshal(report)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(string(data))
	} else {
		fmt.Printf("STATUS: %s\n", status)
		if getReport {
			data := report.AsText(0)
			for _, line := range data {
				fmt.Println(line)
			}
		}
	}
}

// InitGetStatusCommand is an initialiser for get-status
func InitGetStatusCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "get-status",
		Short: "Get status of deployment",
		Long:  "Get status of deployment",
		Run:   getStatus,
	}

	var getJSON, report bool
	run.Flags().BoolVarP(&getJSON, "json", "j", false, "Output JSON")
	run.Flags().BoolVarP(&report, "report", "r", false, "Get human-readable full report")

	return run
}

func getObjectStatus(cmd *cobra.Command, args []string) {
	key := args[0]
	var err error

	var url string
	if len(args) > 0 {
		url = args[0]
	}
	if url == "" {
		url = os.Getenv("KUBERNETES_CLUSTER_URL")
	}

	c, err := client.New(url)
	if err != nil {
		log.Fatal(err)
	}

	sel, err := labels.Parse("")
	if err != nil {
		log.Fatal(err)
	}

	depGraph, err := scheduler.BuildDependencyGraph(c, sel)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Checking for circular dependencies.")
	cycles := scheduler.DetectCycles(depGraph)
	if len(cycles) > 0 {
		message := "Cycles detected, terminating:\n"
		for _, cycle := range cycles {
			keys := make([]string, 0, len(cycle))
			for _, vertex := range cycle {
				keys = append(keys, vertex.Key())
			}
			message = fmt.Sprintf("%sCycle: %s\n", message, strings.Join(keys, ", "))
		}

		log.Fatal(message)
	} else {
		log.Println("No cycles detected.")
	}

	resource := depGraph[key]
	status, err := resource.Resource.Status(nil)

	fmt.Printf("status: '%s', error: '%s'", status, err)
}
func initGetObjectStatusCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "object-status",
		Short: "Get status of object",
		Long:  "Get status of object",
		Run:   getObjectStatus,
	}

	return run
}
