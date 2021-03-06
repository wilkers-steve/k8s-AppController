// Copyright 2016 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/typed/discovery"
	"k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/Mirantis/k8s-AppController/client"
)

// KubernetesRequiredMajorVersion is minimal required major version of Kubernetes cluster
const KubernetesRequiredMajorVersion = 1

// KubernetesRequiredMinorVersion is minimal required minor version of Kubernetes cluster
const KubernetesRequiredMinorVersion = 4

func getFileContents(stream *os.File) string {
	result := ""
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		result += scanner.Text() + "\n"
	}
	return result
}

func createTPRIfNotExists(tpr extensions.ThirdPartyResource, client unversioned.Client) {
	_, err := client.Extensions().ThirdPartyResources().Create(&tpr)
	switch err.(type) {
	case (*errors.StatusError):
		e := err.(*errors.StatusError)
		if e.ErrStatus.Code != 409 {
			log.Fatal(e)
		} else {
			log.Printf("%s already exists, skipping", e.ErrStatus.Details.Name)
		}
	case nil:
		log.Printf("Created %s", tpr.ObjectMeta.Name)
	default:
		log.Fatal(err)
	}
	return
}

func getDependencyFromPath(path string) extensions.ThirdPartyResource {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	var tpr extensions.ThirdPartyResource
	err = json.Unmarshal([]byte(getFileContents(file)), &tpr)
	if err != nil {
		log.Fatal(err)
	}
	return tpr
}

func checkVersion(c unversioned.Client) {
	v, err := discovery.NewDiscoveryClient(c.RESTClient).ServerVersion()
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile("[0-9]+")
	major, err := strconv.Atoi(re.FindString(v.Major))
	if err != nil {
		log.Fatal(err)
	}
	minor, err := strconv.Atoi(re.FindString(v.Minor))
	if err != nil {
		log.Fatal(err)
	}

	if major < KubernetesRequiredMajorVersion || (major == KubernetesRequiredMajorVersion && minor < KubernetesRequiredMinorVersion) {
		log.Fatal(fmt.Errorf("AppController is not compatible with Kubernetes version older than %d.%d", KubernetesRequiredMajorVersion, KubernetesRequiredMinorVersion))
	}

}

func bootstrap(cmd *cobra.Command, args []string) {
	thirdPartyResourcesPath := os.Args[2]

	dependencyTPR := getDependencyFromPath(thirdPartyResourcesPath + "/dependencies.json")
	definitionTPR := getDependencyFromPath(thirdPartyResourcesPath + "/resdefs.json")

	url := os.Getenv("KUBERNETES_CLUSTER_URL")
	config, err := client.GetConfig(url)
	if err != nil {
		log.Fatal(err)
	}

	c := *unversioned.NewOrDie(config)

	checkVersion(c)

	createTPRIfNotExists(dependencyTPR, c)
	createTPRIfNotExists(definitionTPR, c)
}

// Bootstrap is cobra command for bootstrapping AppController, meant to be run in an init container
var Bootstrap = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap AppController",
	Long:  "Create ThirdPartyResources required for AppController pod to function properly",
	Run:   bootstrap,
}
