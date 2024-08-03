package helmInterface

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func createNamespaceIfNotExists(clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func GetKubeConfig() string {
	var kube_config = os.Getenv("KUBECONFIG")
	if kube_config == "" {
		kube_config = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	return kube_config
}
func GetKubernetesClientSet(kube_config string) (*kubernetes.Clientset, error) {
	//creiamo una nuova configurazione di Kubernetes
	conf, err := clientcmd.BuildConfigFromFlags("", kube_config)
	if err != nil {
		log.Println("Error building kubeconfig: ", err.Error())
		return nil, err

	}
	//creiamo un nuovo client di Kubernetes
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Println("Error creating Kubernetes client: ", err.Error())
		return nil, err
	}
	return clientset, nil
}

func GetNewHelmClient(namespace string) (*action.Configuration, error) {
	//creiamo un nuovo clientset di Kubernetes
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		log.Println("Error getting Kubernetes clientset: ", err.Error())
		return nil, err
	}
	err = createNamespaceIfNotExists(clientset, namespace)
	if err != nil {
		log.Println("Error creating namespace: ", err.Error())
		return nil, err
	}
	actions_settings := cli.New()
	actions_settings.KubeConfig = GetKubeConfig()
	//passiamo un puntatore alla struct di actions che puo compiere Helm
	actions := new(action.Configuration)
	if err := actions.Init(kube.GetConfig(GetKubeConfig(), "", namespace), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Println("Error initializing Helm client: ", err.Error())
		return nil, err
	}
	return actions, nil
}

func Install(chart *chart.Chart, values map[string]interface{}, releaseName string, namespace string) error {
	helm_client, err := GetNewHelmClient(namespace)
	if err != nil {
		log.Println("Error getting Helm client: " + err.Error())
		return err
	}
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

func IsReleaseActive(rel_jwt string, namespace string) (bool, error) {
	helm_client, err := GetNewHelmClient(namespace)
	if err != nil {
		log.Println("Error getting Helm client: " + err.Error())
		return false, err
	}
	status := action.NewList(helm_client)
	rels, err := status.Run()
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

func UninstallRelease(rel_jwt string, namespace string) error {
	helm_client, err := GetNewHelmClient(namespace)
	if err != nil {
		log.Println("Error getting Helm client: " + err.Error())
		return err
	}
	uninstall := action.NewUninstall(helm_client)
	_, err = uninstall.Run(rel_jwt)
	if err != nil {
		log.Println("Error uninstalling release: ", err.Error())
		return err
	}
	log.Println("Uninstalled release", rel_jwt)
	return nil
}
