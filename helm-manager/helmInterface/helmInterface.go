package helmInterface

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetKubeConfig() string {
	var kube_config = os.Getenv("KUBECONFIG")
	if kube_config == "" {
		kube_config = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	return kube_config
}
func GetKubernetesClientSet(kube_config string) *kubernetes.Clientset {
	//creiamo una nuova configurazione di Kubernetes
	conf, err := clientcmd.BuildConfigFromFlags("", kube_config)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	//creiamo un nuovo client di Kubernetes
	config, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	return config
}

func GetNewHelmClient() *action.Configuration {
	actions_settings := cli.New()
	//passiamo un puntatore alla struct di actions che puo compiere Helm
	actions := new(action.Configuration)
	if err := actions.Init(actions_settings.RESTClientGetter(), "default", os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	return actions
}

func Install(chart *chart.Chart, values map[string]interface{}, releaseName string, namespace string) error {
	helm_client := GetNewHelmClient()
	newRelease := action.NewInstall(helm_client)
	newRelease.Namespace = namespace
	newRelease.ReleaseName = releaseName
	rel, err := newRelease.Run(chart, values)
	if err != nil {
		fmt.Println("Error installing release: " + err.Error())
		return err
	}
	fmt.Println("Installed" + rel.Name)
	return nil
}

func CreateChart(chart_name string) (*chart.Chart, error) {
	templateFile, err := os.ReadFile("template.yaml")
	if err != nil {
		log.Fatalf("Error reading file: %s", err.Error())
		return nil, err
	}
	mychart := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    chart_name,
			Version: "0.1.0",
		},
		Templates: []*chart.File{
			{Name: "template.yaml", Data: templateFile},
		},
	}
	return mychart, nil
}

func GetReleaseList(helm_client *action.Configuration) ([]*release.Release, error) {
	list := action.NewList(helm_client)
	rels, err := list.Run()
	if err != nil {
		log.Fatalf("Error getting list of releases: %s", err.Error())
		return nil, err
	}
	return rels, nil
}

func IsReleaseActive(releaseName string) (bool, error) {
	status := action.NewList(GetNewHelmClient()) //occhio qua
	rels, err := status.Run()
	if err != nil {
		log.Fatalf("Error getting list of releases: %s", err.Error())
		return false, err
	}
	for _, rel := range rels {
		if rel.Name == releaseName {
			fmt.Println("Release already active")
			return true, nil
		}
	}
	return false, nil
}

func GetValues(jwt string) (map[string]interface{}, error) {
	//leggi values.yaml da file usando le chartutils ufficiali
	values, err := chartutil.ReadValuesFile(fmt.Sprintf("/shared/uploads/%s/values.yaml", jwt))
	if err != nil {
		log.Fatalf("Error reading values file: %s", err.Error())
		return nil, err
	}
	return values.AsMap(), nil
}

func UninstallRelease(rel_jwt string) error {
	uninstall := action.NewUninstall(GetNewHelmClient())
	_, err := uninstall.Run(rel_jwt)
	if err != nil {
		log.Fatalf("Error uninstalling release: %s", err.Error())
		return err
	}
	fmt.Println("Uninstalled release", rel_jwt)
	return nil
}
