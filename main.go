package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

func handler(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
	fmt.Fprintf(w, "hello!")
}

func main() {
	var parameters WhSvrParameters

	// get command line parameters
	flag.IntVar(&parameters.port, "port", 8080, "Webhook server port.")
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	// pair, err := tls.LoadX509KeyPair(parameters.certFile, parameters.keyFile)
	// if err != nil {
	// 	glog.Errorf("Failed to load key pair: %v", err)
	// }

	// whsvr := &WebhookServer{
	// 	server: &http.Server{
	// 		Addr: fmt.Sprintf(":%v", parameters.port),
	// 		//TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	// 	},
	// }

	// // define http server and server handler
	// mux := http.NewServeMux()
	// mux.HandleFunc("/mutate", whsvr.serve)
	// whsvr.server.Handler = mux

	// start webhook server in new routine
	go func() {
		//if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
		// if err := whsvr.server.ListenAndServe(); err != nil {
		// 	glog.Errorf("Failed to listen and serve webhook server: %v", err)
		// }
		http.HandleFunc("/", handler)
		http.ListenAndServe(":8080", nil)
	}()

	glog.Info("Server started")

	// handle interuptions
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	//whsvr.server.Shutdown(context.Background())
}
