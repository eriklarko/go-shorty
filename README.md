# go-shorty
url shrtnr in go. One binary, stores everything in a json file. Easy.

Run
```bash
go build main/go-shorty.go
```
and then 
```bash
./go-shorty
```

The port can be set by using the `-port=X` flag, defaults to 8080
The file to use as persistent storage can be set via the `-redirFile=/path/to/some/file` flag, defaults to `./redirs.json`

You add and overwrite redirections by sending a GET request to `host/add/short=url`,
e.g. `localhost:8080/add/g=www.google.com`

You delete redirections by sending a GET request to `host/delete/short`
e.g. `localhost:8080/delete/g`
