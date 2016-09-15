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

The server is currently not configurable at all :) But it starts at port 80 and writes the redirects to a file in the working directory called `redirs.json`
