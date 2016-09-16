package main

import (
	"fmt"
	"net/http"
	"log"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"flag"
	"path/filepath"
)

var redirFile string
var db map[string]string

func main() {
	flag.StringVar(&redirFile, "redirFile", "redirs.json", "The path to the file to use as persistent storage")
	port := flag.String("port", "8080", "Which port to start the HTTP server on")
	flag.Parse()

	initializeRedirections()
	startHttpServer(*port)
}

func initializeRedirections() {
	var err error
	db, err = readRedirectFile()
	if err != nil {
		log.Panicf("Could not read redir file, %+v", err)
	}
}

func readRedirectFile() (map[string]string, error) {
	absPath, err := filepath.Abs(redirFile)
	if err != nil {
		log.Printf("Could not read absolute path of %s. Everything is fine but I can't tell you exactly where the config file is\n", err)
	}

	log.Printf("Reading redirects from %s\n", absPath)
	if _, err := os.Stat(redirFile); os.IsNotExist(err) {
		log.Printf("%s was not found, starting without any redirects\n", redirFile)
		return make(map[string]string), nil
	}

	b, err := ioutil.ReadFile(redirFile)
	if err != nil {
		return nil, errors.New("Unable to read the redirect file, " + err.Error())
	}

	m := make(map[string]string)
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, errors.New("Unable to parse contents of redirect file, " + err.Error())
	}

	log.Printf("Starting with the following redirs:\n%v\n", string(b))
	return m, nil
}

func startHttpServer(port string) {
	http.HandleFunc("/", routeRequest)
	log.Printf("Starting http server on port %s\n", port)
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		log.Panicf("Error occured in the http server, %v\n", err)
	}
}

func addRedirection(shortName, redirectTo string) error {
	log.Printf("Adding or updating redirect %s -> %s", shortName, redirectTo)
	db[shortName] = redirectTo
	return persistRedirections()
}

func removeRedirection(shortName string) error {
	delete(db, shortName)
	return persistRedirections()
}

func persistRedirections() error {
	b, err := json.MarshalIndent(db, "", "    ")
	if err != nil {
		return errors.New("Unable to marshal the current redirects to JSON, " + err.Error())
	}
	return ioutil.WriteFile(redirFile, b, 0644)
}

func routeRequest(w http.ResponseWriter, r *http.Request) {
	shortName := r.RequestURI
	log.Printf("Got request for %s", shortName)

	if shortName == "/" {
		prefix := r.Host
		fmt.Fprintf(w, "Welcome to go-shorty\n\nTo add a redirect GET to %s/add/short=url\nTo delete GET to %s/delete/short\nGet a list of all redirects, GET to %s/list", prefix, prefix, prefix)
	} else if strings.HasPrefix(shortName, "/add/") {
		assumeRequestIsAddRedir(w, r)
	} else if strings.HasPrefix(shortName, "/delete/") {
		assumeRequestIsRemoveRedir(w, r)
	} else if strings.HasPrefix(shortName, "/list") {
		a, err := ioutil.ReadFile(redirFile)
		if err == nil {
			fmt.Fprint(w, string(a))
		} else {
			log.Printf("Could not list redirects %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	} else {
		assumeRequestIsARedir(w, r)
	}
}

func assumeRequestIsAddRedir(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseFromAndTo(r.RequestURI)
	if err != nil {
		log.Printf("Could not parse add redirect input %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = addRedirection(from, to)
	if err == nil {
		s := fmt.Sprintf( "Successfully added redirect %s -> %s", from, to)
		log.Println(s)
		fmt.Fprintf(w, s)
	} else {
		reply := fmt.Sprintf("Failed adding redirect %s -> %s , %+v", from, to, err)
		log.Println(reply)
		http.Error(w, reply, 500)
	}
}

func parseFromAndTo(rawString string) (string, string, error) {
	a := strings.Split(rawString, "/")
	if len(a) < 2 {
		reply := fmt.Sprintf("Invalid add format, use /add/from=to")
		return "", "", errors.New(reply);
	}

	rawParts := strings.Split(a[2], "=")
	if len(rawParts) < 2 {
		reply := fmt.Sprintf("Invalid add format, use /add/from=to")
		return "", "", errors.New(reply)
	}

	from := rawParts[0]

	i := strings.Index(rawString, from + "=") + len(from + "=")
	to := rawString[i:]
	if !strings.Contains(to, "://") && strings.Contains(to, ":/") {
		to = strings.Replace(to, ":/", "://", 1)
	}
	if !strings.Contains(to, "://") {
		to = "http://" + to
	}

	return from, to, nil
}

func assumeRequestIsRemoveRedir(w http.ResponseWriter, r *http.Request) {
	a := strings.Split(r.URL.Path, "/")
	if len(a) < 2 {
		reply := fmt.Sprintf("Invalid add format, use %s/delete/short", r.Host)
		log.Println(reply)
		http.Error(w, reply, http.StatusBadRequest)
		return
	}

	from := a[2]
	err := removeRedirection(from)
	if err == nil {
		reply := fmt.Sprintf("Successfully deleted redirect %s ", from)
		log.Println(reply)
		fmt.Fprint(w, reply)
	} else {
		reply := fmt.Sprintf("Failed removing redirect %s, %+v", from, err)
		log.Println(reply)
		http.Error(w, reply, 500)
	}
}

func assumeRequestIsARedir(w http.ResponseWriter, r *http.Request) {
	shortName := r.URL.Path[1:]
	found, redirectTo, err := tryFindMatchForShortName(shortName)
	if err != nil {
		log.Printf("Failed looking up match for %s, %+v", shortName, err)
		reply := fmt.Sprintf("Failed looking up match for %s, %+v", shortName, err)
		http.Error(w, reply, 500)

	} else if found {
		log.Printf("Found match %s -> %s", shortName, redirectTo)
		// TODO: Check if the match is something that can be redirected to
		// it e.g, has to start with a valid protocol such as http://
		http.Redirect(w, r, redirectTo, http.StatusFound)

	} else {
		log.Printf("No match for %s found", shortName)
		reply := fmt.Sprintf("No match for %s found", shortName)
		http.Error(w, reply, 404)

	}
}

func tryFindMatchForShortName(shortName string) (bool, string, error) {
	res := db[shortName]
	return res != "", res, nil
}