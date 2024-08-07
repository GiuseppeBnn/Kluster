package httpHandler

import (
	"helm3-manager/models"
	"helm3-manager/redisInterface"
	"helm3-manager/relHandler"
	"log"
	"net/http"
)

var Message = new(models.Message)

func ComposeMiddlewares(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler) // middleware(handler) è una funzione che ritorna un handler
	}
	return handler
}

func CorsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // permette a tutti di fare richieste, da cambiare in produzione
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, referredChart")
		next.ServeHTTP(w, r)
	})
}

// middleware che verifica la presenza di un token jwt e la sua validità
func JwtTokenVerificationHandler(writer http.ResponseWriter, request *http.Request) {
	token := request.Header.Get("Authorization")
	//fmt.Println("New request with auth Token: ", token)
	check, err := redisInterface.CheckPresence(token)
	if err != nil {
		http.Error(writer, Message.JsonError("Error in token verification"), http.StatusInternalServerError)
		log.Println("Error in token verification: ", err.Error())
		return
	}
	if !check {
		http.Error(writer, Message.JsonError("Unauthorized request"), http.StatusUnauthorized)
		log.Println("Unauthorized request from", request.RemoteAddr)
		return
	}
}

// questa funzione rivece una post con un campo name, un file yaml ed un file zip, il file zip non è obbligatorio e se presente deve essere estratto in una cartella
// con nome di un token jwt appena generato
func UploadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		check, err := relHandler.CheckNumberOfReleasePerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, Message.JsonError(err), http.StatusInternalServerError)
			log.Println("Error in file upload: ", err.Error())
			return
		}
		if !check {
			http.Error(w, Message.JsonError("Forbidden request"), http.StatusForbidden)
			log.Println("Forbidden request from", r.RemoteAddr, "for exceeding the number of releases")
			return
		}
		if r.Method == "POST" {
			jwt := relHandler.MakeUnicJwt()
			relHandler.MakeReleaseDirIfNotExist(jwt)
			err := relHandler.ZipHandler(r, jwt)
			err2 := relHandler.YamlHandler(r, jwt)
			err3 := relHandler.SaveToRedis(jwt, r.FormValue("name"), r.Header.Get("Authorization"))

			if err != nil || err2 != nil || err3 != nil {
				http.Error(w, Message.JsonError(err, err2, err3), http.StatusInternalServerError)
				log.Println("Error in file upload: ", err.Error())
				relHandler.RemoveFolderDirectoryIfExist(jwt)
				return
			}
			w.WriteHeader(http.StatusOK)

		}
	})
}

func ListHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json_rels, err := relHandler.GetReleasesList(w, r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, Message.JsonError("Error in getting list"), http.StatusInternalServerError)
				log.Println("Error in getting list: ", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(Message.JsonMessage(json_rels)))
			if err != nil {
				log.Println("Could not write response", err)
			}
		}

	})
}

func InstallHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.InstallRelease(w, r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				log.Println("Error in installing release")
				http.Error(w, Message.JsonError("Error in installing release"), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}

	})
}

func DeleteHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.DeleteRelease(r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				http.Error(w, Message.JsonError("Error in deleting release"), http.StatusInternalServerError)
				log.Println("Error in deleting release: ", err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
		}

	})
}
func StopHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.StopRelease(r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				http.Error(w, Message.JsonError("Error in stopping release"), http.StatusInternalServerError)
				log.Println("Error in stopping release: ", err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	})
}

func DetailsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			details, err := relHandler.GetReleaseDetails(r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				http.Error(w, Message.JsonError("Error in getting details"), http.StatusInternalServerError)
				log.Println("Error in getting details: ", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(Message.JsonMessage(details)))
			if err != nil {
				log.Println("Could not write response", err)
			}
		}
	})
}
func LogsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			logs, err := relHandler.GetReleaseLogs(r.Header.Get("Authorization"), r.Header.Get("referredChart"), r.Header.Get("podName"))
			if err != nil {
				http.Error(w, Message.JsonError("Error in getting logs"), http.StatusInternalServerError)
				log.Println("Error in getting logs: ", err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(Message.JsonMessage(logs)))
			if err != nil {
				log.Println("Could not write response", err)
			}
		}
	})
}
