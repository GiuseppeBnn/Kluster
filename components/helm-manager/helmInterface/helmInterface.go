package helmInterface

import (
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
)

func GetNewHelmClient(namespace string, kube_client_set *kubernetes.Clientset, kube_config string) (*action.Configuration, error) {
	actions_settings := cli.New()
	actions_settings.KubeConfig = kube_config
	//passiamo un puntatore alla struct di actions che puo compiere Helm
	actions := new(action.Configuration)
	if err := actions.Init(kube.GetConfig(kube_config, "", namespace), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Println("Error initializing Helm client: ", err.Error())
		return nil, err
	}
	return actions, nil
}

func Install(chart *chart.Chart, values map[string]interface{}, releaseName string, namespace string, helm_client *action.Configuration) error {
	newRelease := action.NewInstall(helm_client)
	newRelease.Namespace = namespace
	newRelease.ReleaseName = releaseName
	rel, err := newRelease.Run(chart, values)
	if err != nil {
		log.Println("Error installing release: " + err.Error())
		return err
	}
	log.Println("Installed" + rel.Name)
	return nil
}

func CreateChart(chart_name string) (*chart.Chart, error) {
	templateFile, err := os.ReadFile("template.yaml")
	if err != nil {
		log.Println("Error reading template file: ", err.Error())
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
		log.Println("Error getting list of releases: ", err.Error())
		return nil, err
	}
	return rels, nil
}

// TODO: evitare di iterare su tutta la lista di release, cercare per namespace
func IsReleaseActive(rel_jwt string, namespace string, helm_client *action.Configuration) (bool, error) {
	rels, err := GetReleaseList(helm_client)
	if err != nil {
		log.Println("Error getting list of releases: ", err.Error())
		return false, err
	}
	for _, rel := range rels {
		if rel.Name == rel_jwt {
			log.Println("Release already active")
			return true, nil
		}
	}
	return false, nil
}

func GetValues(jwt string) (map[string]interface{}, error) {
	//leggi values.yaml da file usando le chartutils ufficiali
	values, err := chartutil.ReadValuesFile("/shared/uploads/" + jwt + "/values.yaml")
	if err != nil {
		log.Println("Error reading values file: ", err.Error())
		return nil, err
	}
	return values.AsMap(), nil
}

func UninstallRelease(rel_jwt string, namespace string, helm_client *action.Configuration) error {
	uninstall := action.NewUninstall(helm_client)
	_, err := uninstall.Run(rel_jwt)
	if err != nil {
		log.Println("Error uninstalling release: ", err.Error())
		return err
	}
	log.Println("Uninstalled release", rel_jwt)
	return nil
}
