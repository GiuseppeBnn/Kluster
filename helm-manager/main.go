package main

import (
	"fmt"
	"helm3-manager/httpHandler"
	"helm3-manager/relHandler"
	"log"
	"net/http"
)

func main() {
	//helm_client := helmInterface.GetNewHelmClient()
	relHandler.MakeUploadDirIfNotExist()

	jwtVerHandler := http.HandlerFunc(httpHandler.JwtTokenVerificationHandler)
	middlewaresSetForUpload := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.UploadHandler)
	middlewaresSetForList := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.ListHandler)
	middlewaresSetForInstall := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.InstallHandler)
	middlewaresSetForDelete := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.DeleteHandler)
	middlewaresSetForStop := httpHandler.ComposeMiddlewares(jwtVerHandler, httpHandler.StopHandler)

	http.Handle("/upload", middlewaresSetForUpload)
	http.Handle("/list", middlewaresSetForList)
	http.Handle("/install", middlewaresSetForInstall)
	http.Handle("/delete", middlewaresSetForDelete)
	http.Handle("/stop", middlewaresSetForStop)
	fmt.Println("Server started at port 9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
