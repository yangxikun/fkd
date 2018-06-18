package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"flag"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
	"strings"
	"gopkg.in/yaml.v2"
	"github.com/deckarep/golang-set"
	"github.com/golang/glog"
	coreV1 "k8s.io/api/core/v1"
)

var podNamespace = flag.String("pod-namespace", "", "discover pods in namespace")
var selectors = flag.String("pod-selectors", "", "comma separate pod selectors")
var filebeatNamespace = flag.String("filebeat-namespace", "", "filebeat namespace")
var configMap = flag.String("configmap", "", "filebeat prospectors kubernetes configmap")
var configMapKey = flag.String("configmap-key", "", "filebeat prospectors kubernetes configmap key")

var filebeatProspectorsK8sYaml []struct{
	Type string
	ContainersIDs []string `yaml:"containers.ids"`
	Processors []struct{
		AddKuernetesMetadata struct{
			InCluster bool `yaml:"in_cluster"`
		} `yaml:"add_kubernetes_metadata"`
	} `yaml:"processors"`
}

func main() {
	flag.Parse()
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalln(err)
	}
	clientset := kubernetes.NewForConfigOrDie(config)
	tick := time.NewTicker(10*time.Second)
	for {
		<- tick.C
		// list pod
		var podList []coreV1.Pod
		for _, selector := range strings.Split(*selectors, ",") {
			pl, err := clientset.CoreV1().Pods(*podNamespace).List(v1.ListOptions{
				LabelSelector: selector,
			})
			if err != nil {
				glog.Error(err)
				continue
			}
			podList = append(podList, pl.Items...)
		}
		// get pod container id
		var containerIDs []string
		for _, pod := range podList {
			glog.V(1).Infoln(pod.Name)
			for _, status := range pod.Status.ContainerStatuses {
				containerIDs = append(containerIDs, strings.TrimPrefix(status.ContainerID, "docker://"))
			}
		}
		// get configmap
		cm, err := clientset.CoreV1().ConfigMaps(*filebeatNamespace).Get(*configMap, v1.GetOptions{})
		if err != nil {
			glog.Error(err)
			continue
		}
		glog.V(1).Infoln(cm.Data)
		err = yaml.Unmarshal([]byte(cm.Data[*configMapKey]), &filebeatProspectorsK8sYaml)
		if err != nil {
			glog.Error(err)
			continue
		}
		containerIDsSetA := mapset.NewSet()
		containerIDsSetB := mapset.NewSet()
		for _, containerID := range containerIDs {
			containerIDsSetA.Add(containerID)
		}
		for _, containerID := range filebeatProspectorsK8sYaml[0].ContainersIDs {
			containerIDsSetB.Add(containerID)
		}
		if containerIDsSetA.Equal(containerIDsSetB) {
			continue
		}

		// update configmap
		glog.Infoln("update", containerIDs)
		filebeatProspectorsK8sYaml[0].ContainersIDs = containerIDs
		updateYaml, _ := yaml.Marshal(filebeatProspectorsK8sYaml)
		cm.Data[*configMapKey] = string(updateYaml)
		_, err = clientset.CoreV1().ConfigMaps(*filebeatNamespace).Update(cm)
		if err != nil {
			glog.Error(err)
			continue
		}
	}
}
