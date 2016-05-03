// rest
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/elgs/gorest2"
)

func serve() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", r.Header.Get("Access-Control-Request-Method"))
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))

		if r.Method == "OPTIONS" {
			return
		}

		urlPath := r.URL.Path
		var dataHandler func(w http.ResponseWriter, r *http.Request)
		if strings.HasPrefix(urlPath, "/api/") {
			dataHandler = gorest2.GetHandler("/api")
		} else {
			dataHandler = gorest2.GetHandler(urlPath)
		}
		if dataHandler == nil {
			http.Error(w, "Not found.", http.StatusNotFound)
			return
		}
		for _, globalHandlerInterceptor := range gorest2.GlobalHandlerInterceptorRegistry {
			ctn, err := globalHandlerInterceptor.BeforeHandle(w, r)
			if !ctn || err != nil {
				fmt.Fprint(w, err.Error())
				return
			}
		}
		handlerInterceptor := gorest2.HandlerInterceptorRegistry[urlPath]
		if handlerInterceptor != nil {
			ctn, err := handlerInterceptor.BeforeHandle(w, r)
			if !ctn || err != nil {
				fmt.Fprint(w, err.Error())
				return
			}
		}
		dataHandler(w, r)
		if handlerInterceptor != nil {
			err := handlerInterceptor.AfterHandle(w, r)
			if err != nil {
				fmt.Fprint(w, err.Error())
				return
			}
		}
		for _, globalHandlerInterceptor := range gorest2.GlobalHandlerInterceptorRegistry {
			err := globalHandlerInterceptor.AfterHandle(w, r)
			if err != nil {
				fmt.Fprint(w, err.Error())
				return
			}
		}
	}
	//	_ = handler
	http.HandleFunc("/", handler)

	if enableHttp {
		go func() {
			fmt.Println(fmt.Sprint("Listening on http://", hostHttp, ":", portHttp, "/"))
			err := http.ListenAndServe(fmt.Sprint(hostHttp, ":", portHttp), nil)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
	if enableHttps {
		go func() {
			fmt.Println(fmt.Sprint("Listening on https://", hostHttps, ":", portHttps, "/"))
			err := http.ListenAndServeTLS(fmt.Sprint(hostHttps, ":", portHttps), certFile, keyFile, nil)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
}
