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
)

var redirFile string = "redirs.json"
var db map[string]string

func main() {
	var err error
	db, err = readRedirectFile()
	if err != nil {
		log.Panicf("Could not read redir file, %+v", err)
	}

	http.HandleFunc("/", handler)
	err = http.ListenAndServe(":80", nil)
	if err == nil {
		log.Println("Started http server on port 80")
	} else {
		log.Panicf("Could not start http server, %v\n", err)
	}
}

func readRedirectFile() (map[string]string, error) {
	if _, err := os.Stat(redirFile); os.IsNotExist(err) {
		log.Println("No config file found, starting without any redirects")
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

	log.Printf("Starting with the following redirs:\n %v\n", string(b))
	return m, nil
}

func addRedir(shortName, redirectTo string) error {
	log.Printf("Adding or updating redirect %s -> %s", shortName, redirectTo)
	db[shortName] = redirectTo
	return persistRedirs()
}

func removeRedir(shortName string) error {
	delete(db, shortName)
	return persistRedirs()
}

func persistRedirs() error {
	b, err := json.MarshalIndent(db, "", "    ")
	if err != nil {
		return errors.New("Unable to marshal the current redirects to JSON, " + err.Error())
	}
	return ioutil.WriteFile(redirFile, b, 0644)
}

func handler(w http.ResponseWriter, r *http.Request) {
	shortName := r.URL.Path[1:]
	log.Printf("Got request for %s", shortName)

	if len(shortName) == 0{
		fmt.Fprintf(w, "Welcome to go-shorty, to add a redirect GET to %s/add/short=url\nto delete GET to %s/delete/short", r.URL.Host, r.URL.Host)
	} else if strings.HasPrefix(shortName, "add/") {
		assumeRequestIsAddRedir(w, r)
	} else if strings.HasPrefix(shortName, "remove/") {
		assumeRequestIsRemoveRedir(w, r)
	} else {
		assumeRequestIsARedir(w, r)
	}
}

func assumeRequestIsAddRedir(w http.ResponseWriter, r *http.Request) {
	a := strings.Split(r.URL.Path, "/")
	if len(a) < 2 {
		reply := fmt.Sprintf("Invalid add format, use %s/add/from=to", r.Host)
		http.Error(w, reply, http.StatusBadRequest)
		return
	}

	rawParts := strings.Split(a[2], "=")
	if len(rawParts) < 2 {
		reply := fmt.Sprintf("Invalid add format, use %s/add/from=to", r.Host)
		http.Error(w, reply, http.StatusBadRequest)
		return
	}

	from := rawParts[0]

	i := strings.Index(r.URL.Path, from + "=") + len(from + "=")
	to := r.URL.Path[i:]
	if !strings.Contains(to, "://") && strings.Contains(to, ":/") {
		to = strings.Replace(to, ":/", "://", 1)
	}
	if !strings.Contains(to, "://") {
		to = "http://" + to
	}

	err := addRedir(from, to)
	if err == nil {
		fmt.Fprintf(w, "Successfully added redirect %s -> %s", from, to)
	} else {
		reply := fmt.Sprintf("Failed adding redirect %s -> %s , %+v", from, to, err)
		http.Error(w, reply, 500)
	}
}

func assumeRequestIsRemoveRedir(w http.ResponseWriter, r *http.Request) {
	a := strings.Split(r.URL.Path, "/")
	if len(a) < 2 {
		reply := fmt.Sprintf("Invalid add format, use %s/delete/short", r.Host)
		http.Error(w, reply, http.StatusBadRequest)
		return
	}

	from := a[2]
	err := removeRedir(from)
	if err == nil {
		fmt.Fprint(w, "Successfully deleted redirect %s ", from,)
	} else {
		reply := fmt.Sprintf("Failed removing redirect %s, %+v", from, err)
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