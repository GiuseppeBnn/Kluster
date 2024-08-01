package relHandler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"helm3-manager/helmInterface"
	"helm3-manager/redisInterface"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v4"
)

const maxReleasePerUser = 2

func MakeUploadDirIfNotExist() {
	_, err := os.Stat("/shared/uploads")
	if os.IsNotExist(err) {
		os.Mkdir("/shared/uploads", 0755)
	}
}
func adaptToK8s(token string) string { // must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and be less than 53 characters
	var builder strings.Builder
	for _, char := range token {
		if unicode.IsLetter(char) && unicode.IsLower(char) || unicode.IsNumber(char) || char == '-' {
			builder.WriteRune(char)
		} else if unicode.IsLetter(char) && unicode.IsUpper(char) {
			builder.WriteRune(unicode.ToLower(char))
		}
	}

	sanitized := builder.String()
	if len(sanitized) > 50 {
		return sanitized[:50]
	}
	return sanitized
}

// unicJwt must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
func MakeUnicJwt() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte("segretone_da_cambiare"))
	return adaptToK8s(tokenString)

}
func MakeUnicJwtForNamespace(name string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(name))
	return adaptToK8s(tokenString)
}
func makeReleaseDirIfNotExist(jwt string) {
	_, err := os.Stat("/shared/uploads/" + jwt)
	if os.IsNotExist(err) {
		os.Mkdir("/shared/uploads/"+jwt, 0755)
	}
}

// function that return an error if something goes wrong
func ZipHandler(r *http.Request, jwt string) error {
	r.ParseMultipartForm(10 << 20) // max 10 MB
	file, handler, err := r.FormFile("zipFile")
	if err != nil {
		fmt.Println("File not found")
		return err
	}
	defer file.Close()
	if filepath.Ext(handler.Filename) == ".zip" {
		makeReleaseDirIfNotExist(jwt)
		// Legge il contenuto del file caricato in memoria
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println("Could not read file content", err)
			return err
		}
		// Creazione di un reader per il file zip
		zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
		if err != nil {
			fmt.Println("Could not open zip file", err)
			return err
		}

		for _, file := range zipReader.File {
			fileReader, err := file.Open()
			if err != nil {
				fmt.Println("Could not open file in zip")
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
					fmt.Println("Could not create file", err)
					return err
				}
				defer fileToCreate.Close()

				_, err = io.Copy(fileToCreate, fileReader)
				if err != nil {
					fmt.Println("Could not copy file", err)
					return err
				}
			}
		}
	} else {
		fmt.Println("File is not a zip")
		return err
	}
	return nil
}
func YamlHandler(r *http.Request, jwt string) error {
	r.ParseMultipartForm(2 << 20)
	file, handler, err := r.FormFile("yamlFile")
	if err != nil {
		fmt.Println("File not found")
		return err
	}
	defer file.Close()

	// Creazione di un file yaml
	if filepath.Ext(handler.Filename) == ".yaml" {
		fileToCreate, err := os.OpenFile("/shared/uploads/"+jwt+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Println("Could not create file", err)
			return err
		}
		defer fileToCreate.Close()

		_, err = io.Copy(fileToCreate, file)
		if err != nil {
			fmt.Println("Could not copy file", err)
			return err
		}
	} else {
		fmt.Println("File is not a yaml")
		return err
	}
	return nil
}

func SaveToRedis(jwt string, name string, token string) error {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		fmt.Println("Could not get key", err)
		return err
	}
	namespaceJwt := MakeUnicJwtForNamespace(name)
	err = redisInterface.InsertInSet("rel-"+cf, PrepareJsonString(jwt, name, namespaceJwt))
	if err != nil {
		fmt.Println("Could not insert in set", err)
		return err
	}
	return nil
}

func DeleteRelease(token string, jwt string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, jwt)
	if err != nil {
		fmt.Println("Could not get release", err)
		return err
	}
	if rel == nil {
		fmt.Println("Release not found")
		return nil
	}
	// controlla se la release è già attiva
	check, err := helmInterface.IsReleaseActive(jwt)
	if err != nil {
		fmt.Println("Could not check if release is active", err)
		return err
	}
	if check {
		fmt.Println("Release active, cannot delete")
	} else {
		cf, err := redisInterface.GetKeyValue(token)
		if err != nil {
			fmt.Println("Could not get key value", err)
			return err
		}
		rel, err := getReleaseStringFromToken(token, jwt)
		if err != nil {
			fmt.Println("Could not get release", err)
			return err
		}
		if err != nil {
			fmt.Println("Could not Marshal release map", err)
			return err
		}
		err = redisInterface.DeleteFromSet(cf, rel)
		if err != nil {
			fmt.Println("Could not delete from set", err)
			return err
		}
		err = os.RemoveAll("/shared/uploads/" + jwt)
		if err != nil {
			fmt.Println("Could not remove jwt directory", err)
			return err
		}
	}
	return nil
}
func CheckNumberOfReleasePerToken(token string) (bool, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		return false, err
	}
	n, err := redisInterface.GetNumberOfSetFromKey("rel-" + cf)
	if err != nil {
		fmt.Println("Could not get number of release", err)
		return false, err
	}
	return int(n) <= (maxReleasePerUser - 1), nil
}

func PrepareJsonString(jwt string, name string, nsJwt string) string {
	return fmt.Sprintf(`{"jwt": "%s", "name": "%s", "namespace": "%s"}`, jwt, name, nsJwt)
}

func GetReleasesList(w http.ResponseWriter, token string) error {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		http.Error(w, "Error retrieving cf", http.StatusInternalServerError)
		return err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		http.Error(w, "Error retrieving set from Redis", http.StatusInternalServerError)
		return err
	}
	rels, err := checkAndSetActive(val)
	if err != nil {
		http.Error(w, "Error checking if release is active", http.StatusInternalServerError)
		return err
	}
	jsonRels := combineJsonsInArray(rels...)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, jsonRels)
	return nil
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
		fmt.Println("Checking if release is active")

		check, err := helmInterface.IsReleaseActive(json_rel["jwt"].(string))
		if err != nil {
			return nil, err
		}

		if check {
			json_rel["status"] = "active"
		} else {
			json_rel["status"] = "inactive"
		}
		json_bytes, err := json.Marshal(json_rel)
		if err != nil {
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
		return nil, err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		return nil, err
	}
	for _, rel := range val {
		json_rel := make(map[string]interface{})
		json.Unmarshal([]byte(rel), &json_rel)
		if json_rel["jwt"] == jwt {
			return json_rel, nil
		}
		fmt.Println("Release not found")
		fmt.Println(json_rel["jwt"], "|-|", jwt)
	}
	return nil, nil
}

func InstallRelease(w http.ResponseWriter, token string, referredChart string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, referredChart)
	if err != nil {
		http.Error(w, "Error getting release", http.StatusInternalServerError)
		return err
	}
	if rel == nil {
		http.Error(w, "Release not found in your releases", http.StatusNotFound)
		return nil
	}
	// controlla se la release è già attiva
	check, err := helmInterface.IsReleaseActive(rel["jwt"].(string))
	if err != nil {
		http.Error(w, "Error checking if release is active", http.StatusInternalServerError)
		return err
	}
	if check {
		http.Error(w, "Release already active", http.StatusForbidden)
		return nil
	}
	chart, err := helmInterface.CreateChart(referredChart)
	if err != nil {
		http.Error(w, "Error creating chart", http.StatusInternalServerError)
		return err
	}
	values, err := getValuesMapFromToken(referredChart)
	if err != nil {
		http.Error(w, "Error getting values", http.StatusInternalServerError)
		return err
	}
	err = helmInterface.Install(chart, values, rel["jwt"].(string), rel["namespace"].(string))
	if err != nil {
		http.Error(w, "Error installing release", http.StatusInternalServerError)
		return err
	}
	return nil
}

func getValuesMapFromToken(rel_jwt string) (map[string]interface{}, error) {
	//leggi values.yaml da file usando le chartutils ufficiali
	values, err := helmInterface.GetValues(rel_jwt)
	if err != nil {
		return nil, err
	}
	values["rootDirectory"] = fmt.Sprintf("/shared/uploads/%s/mnt/", rel_jwt)
	fmt.Println("rootDirectory: ", values["rootDirectory"])
	return values, nil

}

/*func printUnmarshalledValues(values map[string]interface{}) {
	marshalled_values, err := json.Marshal(values)
	if err != nil {
		fmt.Println("Could not marshal values", err)
	}
	fmt.Println("Values: ", string(marshalled_values))
	fmt.Printf("il tipo di values è %T\n", values)

}*/

func getReleaseStringFromToken(token string, jwt string) (string, error) {
	cf, err := redisInterface.GetKeyValue(token)
	if err != nil {
		return "", err
	}
	val, err := redisInterface.GetAllSetFromKey("rel-" + cf)
	if err != nil {
		return "", err
	}
	for _, rel := range val {
		json_rel := make(map[string]interface{})
		json.Unmarshal([]byte(rel), &json_rel)
		if json_rel["jwt"] == jwt {
			return rel, nil
		}
		fmt.Println("Release not found")
		fmt.Println(json_rel["jwt"], "|-|", jwt)
	}
	return "", nil
}

func StopRelease(token string, referredChart string) error {
	// controlla se la release appartiene all'utente che richiede l'installazione
	rel, err := getReleaseFromToken(token, referredChart)
	if err != nil {
		fmt.Println("Could not get release", err)
		return err
	}
	if rel == nil {
		fmt.Println("Release not found")
		return nil
	}
	// controlla se la release è già attiva
	check, err := helmInterface.IsReleaseActive(rel["jwt"].(string))
	if err != nil {
		fmt.Println("Could not check if release is active", err)
		return err
	}
	if !check {
		fmt.Println("Release not active")
		return nil
	}
	err = helmInterface.UninstallRelease(rel["jwt"].(string))
	if err != nil {
		fmt.Println("Could not uninstall release", err)
		return err
	}
	return nil
}
