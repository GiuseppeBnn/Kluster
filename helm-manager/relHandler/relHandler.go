package relHandler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"helm3-manager/helmInterface"
	"helm3-manager/k8sInterface"
	"helm3-manager/redisInterface"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v4"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/client-go/kubernetes"
)

const maxReleasePerUser = 2
const secretForJwt = "segretone_da_cambiare"

func RemoveFolderDirectoryIfExist(jwt string) error {
	err := os.RemoveAll("/shared/uploads/" + jwt)
	if err != nil {
		log.Println("Could not remove jwt directory", err)
		return err
	}
	return nil
}

func checkZipFilePresence(r *http.Request) (multipart.File, *multipart.FileHeader, bool) {
	r.ParseMultipartForm(10 << 20) // max 10 MB
	file, header, err := r.FormFile("zipFile")
	if err != nil {
		log.Println("File not found")
		return nil, nil, false
	}
	return file, header, true
}

func MakeUploadDirIfNotExist() error {
	_, err := os.Stat("/shared/uploads")
	if os.IsNotExist(err) {
		os.Mkdir("/shared/uploads", 0755)
	}
	return nil
}
func adaptToK8s(token string) string { // must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and be less than 53 characters
	var builder strings.Builder
	for _, char := range token {
		if (unicode.IsLetter(char) && unicode.IsLower(char)) || unicode.IsNumber(char) {
			builder.WriteRune(char)
		} else if unicode.IsLetter(char) && unicode.IsUpper(char) {
			builder.WriteRune(unicode.ToLower(char))
		}
	}
	sanitized := builder.String()
	if len(sanitized) > 50 {
		return sanitized[len(sanitized)-50:]
	}
	return sanitized
}

// unicJwt must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
func MakeUnicJwt() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(time.Now().String() + secretForJwt))
	return adaptToK8s(tokenString)
}
func MakeUnicJwtForNamespace(name string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(time.Now().String() + name))
	return adaptToK8s(tokenString)
}
func MakeReleaseDirIfNotExist(jwt string) {
	_, err := os.Stat("/shared/uploads/" + jwt)
	if os.IsNotExist(err) {
		os.Mkdir("/shared/uploads/"+jwt, 0755)
	}
}

func ZipHandler(r *http.Request, jwt string) error {
	file, handler, presence := checkZipFilePresence(r)
	if !presence {
		return nil
	}
	defer file.Close()
	if filepath.Ext(handler.Filename) == ".zip" {
		// Legge il contenuto del file caricato in memoria
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			log.Println("Could not read file content", err)
			return err
		}
		// Creazione di un reader per il file zip
		zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
		if err != nil {
			log.Println("Could not open zip file", err)
			return err
		}

		for _, file := range zipReader.File {
			fileReader, err := file.Open()
			if err != nil {
				log.Println("Could not open file in zip")
				return err
			}
			defer fileReader.Close()
			//caso in cui il file è una directory
			if file.FileInfo().IsDir() {
				os.MkdirAll("/shared/uploads/"+jwt+"/mnt/"+file.Name, file.Mode())
			} else {
				//caso in cui il file è un file
				fileToCreate, err := os.OpenFile("/shared/uploads/"+jwt+"/mnt/"+file.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
				if err != nil {
					log.Println("Could not create file", err)
					return err
				}
				defer fileToCreate.Close()

				_, err = io.Copy(fileToCreate, fileReader)
				if err != nil {
					log.Println("Could not copy file", err)
					return err
				}
			}
		}
	} else {
		return fmt.Errorf("file is not a zip")
	}
	return nil
}
func YamlHandler(r *http.Request, jwt string) error {
	r.ParseMultipartForm(2 << 20)
	file, handler, err := r.FormFile("yamlFile")
	if err != nil {
		log.Println("File not found")
		return err
	}
	defer file.Close()

	// Creazione di un file yaml
	if filepath.Ext(handler.Filename) == ".yaml" {
		fileToCreate, err := os.OpenFile("/shared/uploads/"+jwt+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Println("Could not create file", err)
			return err
		}
		defer fileToCreate.Close()

		_, err = io.Copy(fileToCreate, file)
		if err != nil {
			log.Println("Could not copy file", err)
			return err
		}
	} else {
		log.Println("File is not a yaml")
		return err
	}
	return nil
}

func SaveToRedis(jwt string, name string, token string) error {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		log.Println("Could not get key", err)
		return err
	}
	namespaceJwt := MakeUnicJwtForNamespace(name)
	err = redisInterface.InsertInSet("rel-"+cf, PrepareJsonString(jwt, name, namespaceJwt))
	if err != nil {
		log.Println("Could not insert in set", err)
		return err
	}
	return nil
}

func DeleteRelease(token string, jwt string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, jwt)
	if err != nil {
		log.Println("Could not get release", err)
		return err
	}
	if rel == nil {
		log.Println("Release not found")
		return nil
	}
	kube_config := k8sInterface.GetKubeConfig()
	kube_client_set, err := k8sInterface.GetKubernetesClientSet(kube_config)
	if err != nil {
		log.Println("Could not get Kubernetes client", err)
		return err
	}
	helm_client, err := helmInterface.GetNewHelmClient(rel["namespace"].(string), kube_client_set, kube_config)
	if err != nil {
		log.Println("Could not get Helm client", err)
		return err
	}
	// controlla se la release è già attiva, TODO: possibile dividere in due funzioni
	check, err := helmInterface.IsReleaseActive(jwt, rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not check if release is active", err)
		return err
	}
	if check {
		log.Println("Release active, cannot delete")
	} else {
		cf, err := redisInterface.GetKeyValue(token)
		if err != nil {
			log.Println("Could not get key value", err)
			return err
		}
		rel, err := getReleaseStringFromToken(token, jwt)
		if err != nil {
			log.Println("Could not get release", err)
			return err
		}
		err = redisInterface.DeleteFromSet(cf, rel)
		if err != nil {
			log.Println("Could not delete from set", err)
			return err
		}
		err = os.RemoveAll("/shared/uploads/" + jwt)
		if err != nil {
			log.Println("Could not remove jwt directory", err)
			return err
		}
	}
	return nil
}
func CheckNumberOfReleasePerToken(token string) (bool, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		log.Println("Could not get key value", err)
		return false, err
	}
	n, err := redisInterface.GetNumberOfSetFromKey("rel-" + cf)
	if err != nil {
		log.Println("Could not get number of release", err)
		return false, err
	}
	return int(n) <= (maxReleasePerUser - 1), nil
}

func PrepareJsonString(jwt string, name string, nsJwt string) string {
	return fmt.Sprintf(`{"jwt": "%s", "name": "%s", "namespace": "%s"}`, jwt, name, nsJwt)
}

func GetReleasesList(w http.ResponseWriter, token string) (string, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		log.Println("Could not get cf", err)
		http.Error(w, "Error retrieving cf", http.StatusInternalServerError)
		return "", err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		log.Println("Could not get set from Redis", err)
		http.Error(w, "Error retrieving set from Redis", http.StatusInternalServerError)
		return "", err
	}
	rels, err := checkAndSetActive(val)
	if err != nil {
		log.Println("Could not check if release is active", err)
		http.Error(w, "Error checking if release is active", http.StatusInternalServerError)
		return "", err
	}
	jsonRels := combineJsonsInArray(rels...)

	return jsonRels, nil
}

func combineJsonsInArray(jsons ...string) string {
	res := "["
	for i, j := range jsons {
		res += j
		if i != len(jsons)-1 {
			res += ","
		}
	}
	res += "]"
	return res
}

/*func getReleasesFromCF(cf string) ([]string, error) {
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		return nil, err
	}
	return val, nil
}*/

func checkAndSetActive(rels []string) ([]string, error) {
	json_rel := make(map[string]interface{})
	checked_rels := make([]string, 0)
	for _, rel := range rels {
		json.Unmarshal([]byte(rel), &json_rel)
		helm_client, err := getHelmClientForNamespace(json_rel["namespace"].(string))
		if err != nil {
			log.Println("Could not get Helm client", err)
			return nil, err
		}
		check, err := helmInterface.IsReleaseActive(json_rel["jwt"].(string), json_rel["namespace"].(string), helm_client)
		if err != nil {
			log.Println("Could not check if release is active", err)
			return nil, err
		}

		if check {
			json_rel["status"] = "active"
		} else {
			json_rel["status"] = "inactive"
		}
		json_bytes, err := json.Marshal(json_rel)
		if err != nil {
			log.Println("Could not marshal json", err)
			return nil, err
		}
		rel = string(json_bytes)
		checked_rels = append(checked_rels, rel)
	}
	return checked_rels, nil
}

func getReleaseFromToken(token string, jwt string) (map[string]interface{}, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		log.Println("Could not get key value", err)
		return nil, err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		log.Println("Could not get set from Redis", err)
		return nil, err
	}
	for _, rel := range val {
		json_rel := make(map[string]interface{})
		json.Unmarshal([]byte(rel), &json_rel)
		if json_rel["jwt"] == jwt {
			return json_rel, nil
		}
		log.Println("Release " + jwt + " not found")
	}
	return nil, nil
}

// TODO: controllare gli http error molto probabilmente non corretti
func InstallRelease(w http.ResponseWriter, token string, referredChart string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, referredChart)
	if err != nil {
		log.Println("Could not get release", err)
		http.Error(w, "Error getting release", http.StatusInternalServerError)
		return err
	}
	if rel == nil {
		log.Println("Release not found")
		http.Error(w, "Release not found in your releases", http.StatusNotFound)
		return nil
	}
	helm_client, err := getHelmClientForNamespace(rel["namespace"].(string))
	if err != nil {
		log.Println("Could not get Helm client", err)
		http.Error(w, "Error getting Helm client", http.StatusInternalServerError)
		return err
	}
	// controlla se la release è già attiva
	check, err := helmInterface.IsReleaseActive(rel["jwt"].(string), rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not check if release is active", err)
		http.Error(w, "Error checking if release is active", http.StatusInternalServerError)
		return err
	}
	if check {
		log.Println("Release " + referredChart + " already active")
		http.Error(w, "Release already active", http.StatusForbidden)
		return nil
	}
	chart, err := helmInterface.CreateChart(referredChart)
	if err != nil {
		log.Println("Could not create chart", err)
		http.Error(w, "Error creating chart", http.StatusInternalServerError)
		return err
	}
	values, err := getValuesMapFromToken(referredChart)
	if err != nil {
		log.Println("Could not get values", err)
		http.Error(w, "Error getting values", http.StatusInternalServerError)
		return err
	}
	err = k8sInterface.CreateNamespaceIfNotExists(rel["namespace"].(string))
	if err != nil {
		log.Println("Error creating namespace: ", err.Error())
		return err
	}
	err = helmInterface.Install(chart, values, rel["jwt"].(string), rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not install release", err)
		http.Error(w, "Error installing release", http.StatusInternalServerError)
		return err
	}
	return nil
}

func getValuesMapFromToken(rel_jwt string) (map[string]interface{}, error) {
	//leggi values.yaml da file usando le chartutils ufficiali
	values, err := helmInterface.GetValues(rel_jwt)
	if err != nil {
		log.Println("Could not get values", err)
		return nil, err
	}
	values["rootDirectory"] = fmt.Sprintf("/shared/uploads/%s/mnt/", rel_jwt)
	return values, nil
}

func getReleaseStringFromToken(token string, jwt string) (string, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		log.Println("Could not get key value", err)
		return "", err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		log.Println("Could not get set from Redis", err)
		return "", err
	}
	for _, rel := range val {
		json_rel := make(map[string]string)
		json.Unmarshal([]byte(rel), &json_rel)
		temp := json_rel["jwt"]
		if temp == jwt {
			return rel, nil
		}
	}
	log.Println("Release " + jwt + " not found")
	return "", fmt.Errorf("release not found")
}

func StopRelease(token string, referredChart string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, referredChart)
	if err != nil {
		log.Println("Could not get release", err)
		return err
	}
	if rel == nil {
		log.Println("Release not found")
		return nil
	}
	helm_client, err := getHelmClientForNamespace(rel["namespace"].(string))
	if err != nil {
		log.Println("Could not get Helm client", err)
		return err
	}
	// controlla se la release è già attiva
	check, err := helmInterface.IsReleaseActive(rel["jwt"].(string), rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not check if release is active", err)
		return err
	}
	if !check {
		log.Println("Release not active")
		return nil
	}
	err = helmInterface.UninstallRelease(rel["jwt"].(string), rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not uninstall release", err)
		return err
	}
	return nil
}

func getK8sClientSetAndConfig() (*kubernetes.Clientset, string, error) {
	kube_config := k8sInterface.GetKubeConfig()
	kube_client_set, err := k8sInterface.GetKubernetesClientSet(kube_config)
	if err != nil {
		log.Println("Could not get Kubernetes client", err)
		return nil, "", err
	}
	return kube_client_set, kube_config, nil
}

func getHelmClientForNamespace(namespace string) (*action.Configuration, error) {
	kube_client_set, kube_config, err := getK8sClientSetAndConfig()
	if err != nil {
		log.Println("Could not get Kubernetes client", err)
		return nil, err
	}
	helm_client, err := helmInterface.GetNewHelmClient(namespace, kube_client_set, kube_config)
	if err != nil {
		log.Println("Could not get Helm client", err)
		return nil, err
	}
	return helm_client, nil
}

func GetReleaseDetails(token string, jwt string) (string, error) {
	json_rel, err := getReleaseFromToken(token, jwt)
	if err != nil {
		log.Println("Could not get release", err)
		return "", err
	}
	helm_client, err := getHelmClientForNamespace(json_rel["namespace"].(string))
	if err != nil {
		log.Println("Could not get Helm client", err)
		return "", err
	}
	check, err := helmInterface.IsReleaseActive(json_rel["jwt"].(string), json_rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not check if release is active", err)
		return "", err
	}
	if check {
		json_rel["status"] = "active"
	} else {
		json_rel["status"] = "inactive"
	}
	details, err := k8sInterface.GetDeploymentsDetails(json_rel["namespace"].(string))
	if err != nil {
		log.Println("Could not get deployments details", err)
		return "", err
	}
	json_rel["details"] = details
	json_bytes, err := json.Marshal(json_rel)
	if err != nil {
		log.Println("Could not marshal json", err)
		return "", err
	}
	return string(json_bytes), nil
}

func GetReleaseLogs(token string, jwt string, podName string) (string, error) {
	json_rel, err := getReleaseFromToken(token, jwt)
	if err != nil {
		log.Println("Could not get release", err)
		return "", err
	}
	helm_client, err := getHelmClientForNamespace(json_rel["namespace"].(string))
	if err != nil {
		log.Println("Could not get Helm client", err)
		return "", err
	}
	check, err := helmInterface.IsReleaseActive(json_rel["jwt"].(string), json_rel["namespace"].(string), helm_client)
	if err != nil {
		log.Println("Could not check if release is active", err)
		return "", err
	}
	if !check {
		log.Println("Release not active")
		return "", nil
	}
	logs, err := k8sInterface.GetLogsFromPods(json_rel["namespace"].(string), podName)
	if err != nil {
		log.Println("Could not get pod logs", err)
		return "", err
	}
	json_rel["logs"] = logs
	json_bytes, err := json.Marshal(json_rel)
	if err != nil {
		log.Println("Could not marshal json", err)
		return "", err
	}
	return string(json_bytes), nil
}

func GetReleaseFromCf(cf string, rel_token string) (string, error) {
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		log.Println("Could not get set from Redis", err)
		return "", err
	}
	for _, rel := range val {
		json_rel := make(map[string]interface{})
		json.Unmarshal([]byte(rel), &json_rel)
		if json_rel["jwt"] == rel_token {
			return rel, nil
		}
	}
	return "", fmt.Errorf("release not found")
}

func DeliverRelease(token string, referredChart string) error {
	rel, err := getReleaseStringFromToken(token, referredChart)
	if err != nil {
		log.Println("Could not get release", err)
		return err
	}
	if rel == "" {
		log.Println("Release not found")
		return nil
	}
	err = redisInterface.InsertInSet("rel-admin", rel)
	if err != nil {
		log.Println("Could not insert in set", err)
		return err
	}
	return nil
}

func UndeliverRelease(token string, referredChart string) error {
	log.Println("Undeliver release", token, referredChart)
	rel, err := getReleaseStringFromToken(token, referredChart)
	if err != nil {
		log.Println("Could not get release", err)
		return err
	}
	if rel == "" {
		log.Println("Release not found")
		return nil
	}
	err = redisInterface.DeleteFromSet("admin", rel)
	if err != nil {
		log.Println("Could not delete from set", err)
		return err
	}
	return nil
}
