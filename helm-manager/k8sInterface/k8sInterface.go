package k8sInterface

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/apps/v1"
	v1n "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func CreateNamespaceIfNotExists(namespace string) error {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		ns := &v1n.Namespace{
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

func RemoveNamespaceIfExists(namespace string) error {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return err
	}
	err = clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	if err != nil {
		return err
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
	conf, err := clientcmd.BuildConfigFromFlags("", kube_config)
	if err != nil {
		log.Println("Error building kubeconfig: ", err.Error())
		return nil, err

	}
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Println("Error creating Kubernetes client: ", err.Error())
		return nil, err
	}
	return clientset, nil
}

func GetDeploymentsFromNamespace(namespace string) (*v1.DeploymentList, error) {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return nil, err
	}
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Println("Error getting deployments: ", err.Error())
		return nil, err
	}
	return deployments, nil
}
func GetDeploymentFromNamespace(namespace string, deploymentName string) (*v1.Deployment, error) {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return nil, err
	}
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		log.Println("Error getting deployment: ", err.Error())
		return nil, err
	}
	return deployment, nil
}

func GetServicesFromDeployment(namespace string, deploymentName string) (*v1n.ServiceList, error) {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return nil, err
	}
	services, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=" + deploymentName,
	})
	if err != nil {
		log.Println("Error getting services: ", err.Error())
		return nil, err
	}
	return services, nil
}

func GetPodsFromDeployment(namespace string, deploymentName string) (*v1n.PodList, error) {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return nil, err
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=" + deploymentName,
	})
	if err != nil {
		log.Println("Error getting pods: ", err.Error())
		return nil, err
	}
	return pods, nil
}

func GetLogsFromPods(namespace string, podName string) (string, error) {
	clientset, err := GetKubernetesClientSet(GetKubeConfig())
	if err != nil {
		return "", err
	}
	podLogOptions := v1n.PodLogOptions{}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetPortsFromDeployment(namespace string, deploymentName string) ([]map[string]interface{}, error) {
	services, err := GetServicesFromDeployment(namespace, deploymentName)
	if err != nil {
		return nil, err
	}
	ports := make([]map[string]interface{}, 0)
	for _, service := range services.Items {
		for _, port := range service.Spec.Ports {
			if service.Spec.Type == "NodePort" {
				ports = append(ports, map[string]interface{}{
					"port":     port.Port,
					"target":   port.TargetPort.IntVal,
					"nodePort": port.NodePort,
				})
				continue
			}
			ports = append(ports, map[string]interface{}{
				"port":   port.Port,
				"target": port.TargetPort.IntVal,
			})
		}
	}
	return ports, nil
}

func GetDeploymentsDetails(namespace string) ([]map[string]interface{}, error) {
	deployments, err := GetDeploymentsFromNamespace(namespace)
	if err != nil {
		return nil, err
	}
	deploymentsDetails, err := extractDetails(namespace, deployments.Items...)
	if err != nil {
		return nil, err
	}
	return deploymentsDetails, nil
}

func GetDeploymentDetails(namespace string, deploymentName string) (map[string]interface{}, error) {
	deployment, err := GetDeploymentFromNamespace(namespace, deploymentName)
	if err != nil {
		return nil, err
	}
	deploymentsDetails, err := extractDetails(namespace, *deployment)
	if err != nil {
		return nil, err
	}
	return deploymentsDetails[0], nil
}

func extractDetails(namespace string, deployments ...v1.Deployment) ([]map[string]interface{}, error) {
	deploymentsDetails := make([]map[string]interface{}, 0)
	for _, deployment := range deployments {
		deploymentDetails := make(map[string]interface{})
		deploymentDetails["name"] = deployment.Name
		ports, err := GetPortsFromDeployment(namespace, deployment.Name)
		deploymentDetails["ports"] = ports
		if err != nil {
			return nil, err
		}
		deploymentDetails["pods"], err = GetPodsFromDeployment(namespace, deployment.Name)
		if err != nil {
			return nil, err
		}
		deploymentDetails["services"], err = GetServicesFromDeployment(namespace, deployment.Name)
		if err != nil {
			return nil, err
		}
		deploymentsDetails = append(deploymentsDetails, deploymentDetails)
	}
	return deploymentsDetails, nil
}
