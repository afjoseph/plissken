package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

var (
	spaBuildPathFlag = flag.String(
		"spa-build-path", "", "path to the build directory of the SPA app")
	portFlag      = flag.String("port", "8080", "port to listen on")
	gitCommitHash = "unknown"
)

func main() {
	flag.Parse()
	if *spaBuildPathFlag == "" {
		log.Fatal("--spa-build-path is required")
	}
	spaBuildPath, err := filepath.Abs(*spaBuildPathFlag)
	if err != nil {
		log.Fatal(err)
	}

	spa := spaHandler{
		staticPath: spaBuildPath,
		indexPath:  filepath.Join(spaBuildPath, "index.html"),
	}
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, gitCommitHash)
	})
	router.PathPrefix("/").Handler(spa)
	srv := &http.Server{
		Handler:      router,
		Addr:         ":" + *portFlag,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
