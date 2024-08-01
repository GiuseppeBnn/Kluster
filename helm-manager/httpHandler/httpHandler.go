package httpHandler

import (
	"fmt"
	"helm3-manager/redisInterface"
	"helm3-manager/relHandler"
	"net/http"
)

func ComposeMiddlewares(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler) // middleware(handler) è una funzione che ritorna un handler
	}
	return handler
}

// middleware che verifica la presenza di un token jwt e la sua validità
func JwtTokenVerificationHandler(writer http.ResponseWriter, request *http.Request) {
	token := request.Header.Get("Authorization")
	//fmt.Println("New request with auth Token: ", token)
	check, err := redisInterface.CheckPresence(token)
	if err != nil {
		http.Error(writer, "Error in token verification", http.StatusInternalServerError)
		return
	}
	if !check {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}
}

// questa funzione rivece una post con un campo name, un file yaml ed un file zip, il file zip non è obbligatorio e se presente deve essere estratto in una cartella
// con nome di un token jwt appena generato
func UploadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		check, err := relHandler.CheckNumberOfReleasePerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, "Error in file upload", http.StatusInternalServerError)
			return
		}
		if !check {
			http.Error(w, "You have uploaded already 2 releases", http.StatusForbidden)
			return
		}
		if r.Method == "POST" {
			jwt := relHandler.MakeUnicJwt()
			//fmt.Println("New upload request with jwt: ", jwt)
			err := relHandler.ZipHandler(r, jwt)
			if err != nil {
				http.Error(w, "Error in file upload", http.StatusInternalServerError)
				return
			}
			err = relHandler.YamlHandler(r, jwt)
			if err != nil {
				http.Error(w, "Error in file upload", http.StatusInternalServerError)
				return
			}
			err = relHandler.SaveToRedis(jwt, r.FormValue("name"), r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, "Error in file upload", http.StatusInternalServerError)
				return
			}

		}
		next.ServeHTTP(w, r)
	})
}

func ListHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.GetReleasesList(w, r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, "Error in getting list", http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func InstallHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.InstallRelease(w, r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				fmt.Println("Error in installing release")
				http.Error(w, "Error in installing release", http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func DeleteHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Println("Deleting release")
			err := relHandler.DeleteRelease(r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				http.Error(w, "Error in deleting release", http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func StopHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			err := relHandler.StopRelease(r.Header.Get("Authorization"), r.Header.Get("referredChart"))
			if err != nil {
				http.Error(w, "Error in stopping release", http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
