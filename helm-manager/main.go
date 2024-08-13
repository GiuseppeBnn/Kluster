package main

import (
	"helm3-manager/httpHandler"
	"helm3-manager/relHandler"
	"log"
	"net/http"
)

const listenPort = ":9000"

func main() {
	relHandler.MakeUploadDirIfNotExist()

	jwtVerHandler := http.HandlerFunc(httpHandler.JwtTokenVerificationHandler)
	middlewaresSetForUpload := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.UploadHandler)
	middlewaresSetForList := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.ListHandler)
	middlewaresSetForInstall := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.InstallHandler)
	middlewaresSetForDelete := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.DeleteHandler)
	middlewaresSetForStop := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.StopHandler)
	middlewaresSetForDetails := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.DetailsHandler)
	middlewaresSetForLogs := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.LogsHandler)
	middlewaresSetForDeliveredList := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.DeliveredListHandler)
	middlewaresSetForUndelivery := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.CorsHandler, httpHandler.UndeliverHandler)

	http.Handle("/upload", middlewaresSetForUpload)
	http.Handle("/list", middlewaresSetForList)
	http.Handle("/install", middlewaresSetForInstall)
	http.Handle("/delete", middlewaresSetForDelete)
	http.Handle("/stop", middlewaresSetForStop)
	http.Handle("/details", middlewaresSetForDetails)
	http.Handle("/logs", middlewaresSetForLogs)
	http.Handle("/delivered", middlewaresSetForDeliveredList)
	http.Handle("/undeliver", middlewaresSetForUndelivery)
	log.Println("Server started at port " + listenPort)
	log.Fatal(http.ListenAndServe(listenPort, nil))
}
