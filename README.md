# Wiki Searcher

## Overview

Wiki searcher is a sample full text search engine (reverse indexed) project written in Go which consist of the following packages.

- **engine** package has a (reversed indexed) indexer which is responsible to index Wiki XML documents by tokenizing the abstract and title of the Wiki document by performing filtering (lower-casing and removing stop words) and stemming phase into the tokens.
- **tcpserver** package has a basic tcp server that is responsible to download and decompress the [Wiki XML dumps](https://dumps.wikimedia.org/) and initialize the indexer with provided document for indexing phase. After these steps tcp server accepts tcp connections and returns the query results.
- **tcpclient** package has a basic tcp client which makes tcp connections to the tcp server for the queries.
- **apiserver** package has a basic REST api server to perform queries over the engine. 

## Basic usage

### Backend

- Cloning the project
```
chasank@development:~/go/src$ git clone https://github.com/xkmsoft/wikisearcher
Cloning into 'wikisearcher'...
remote: Enumerating objects: 123, done.
remote: Counting objects: 100% (123/123), done.
remote: Compressing objects: 100% (86/86), done.
remote: Total 123 (delta 56), reused 95 (delta 31), pack-reused 0
Receiving objects: 100% (123/123), 33.69 KiB | 605.00 KiB/s, done.
Resolving deltas: 100% (56/56), done.
```

- Installing necessary go modules with go mod tidy
```
chasank@development:~/go/src/wikisearcher$ go mod tidy
go: downloading github.com/gorilla/mux v1.8.0
go: downloading github.com/kljensen/snowball v0.6.0
go: downloading github.com/tamerh/xml-stream-parser v1.4.0
go: downloading github.com/RoaringBitmap/roaring v0.9.4
go: downloading github.com/tamerh/xpath v1.0.0
go: downloading github.com/antchfx/xpath v1.2.0
go: downloading github.com/mschoch/smat v0.2.0
go: downloading github.com/bits-and-blooms/bitset v1.2.0
go: downloading github.com/stretchr/testify v1.4.0
go: downloading gopkg.in/yaml.v2 v2.2.2
go: downloading github.com/pmezard/go-difflib v1.0.0
go: downloading github.com/davecgh/go-spew v1.1.0
```

- Initializing the indexer and the tcp server
```
chasank@development:~/go/src/wikisearcher$ go run cmd/engine/main.go 
Initializing the full text search engine and the tcpserver on tcp localhost:3333
Downloading the file from https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract1.xml.gz
Downloading wikimedia dump on https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract1.xml.gz took 39.687648 seconds
Uncompressing the file: data/enwiki-latest-abstract1.xml.gz
Uncompressing the file took 2.553127 seconds
Parsing XML file took 13.956953 seconds
There are 633843 documents in the file data/enwiki-latest-abstract1.xml
Indexing documents took 12.075528 seconds
Saving data dump into the file took 1.328548 seconds
Saving indexes dump into the file took 1.690070 seconds
Whole process took 27.722666 seconds
Initializing the server took 69.963527 seconds
Accepting connections on tcp localhost:3333
```

- Initializing the REST API server
```
chasank@development:~/go/src/wikisearcher$ go run cmd/api/main.go 
API listening connection on :3000

```

### Frontend

- Cloning the [frontend project](https://github.com/xkmsoft/wikiui).
```
chasank@development:~/WebstormProjects$ git clone https://github.com/xkmsoft/wikiui
Cloning into 'wikiui'...
remote: Enumerating objects: 76, done.
remote: Counting objects: 100% (76/76), done.
remote: Compressing objects: 100% (52/52), done.
remote: Total 76 (delta 17), reused 70 (delta 11), pack-reused 0
Unpacking objects: 100% (76/76), 289.36 KiB | 712.00 KiB/s, done.
```

- Installing the necessary npm packages
```
chasank@development:~/WebstormProjects/wikiui$ npm install
```

- Install the following package explicitly. (It looks like I forgot to include the following npm package into the packages.json)
```
chasank@development:~/WebstormProjects/wikiui$ npm install --save @popperjs/core
```

- Running the development server
```
chasank@development:~/WebstormProjects/wikiui$ npm run serve

> wikiui@0.1.0 serve /home/chasank/WebstormProjects/wikiui
> vue-cli-service serve

 INFO  Starting development server...
10% building 2/2 modules 0 active[HPM] Proxy created: [Function: context]  ->  http://localhost:3000
[HPM] Subscribed to http-proxy events:  [ 'error', 'proxyReq', 'close' ]
98% after emitting CopyPlugin

 DONE  Compiled successfully in 4026ms                                                                                                                                                                          2:16:55 PM


  App running at:
  - Local:   http://localhost:8080/ 
  - Network: http://192.168.1.148:8080/

  Note that the development build is not optimized.
  To create a production build, run npm run build.

No issues found.

```
- And finally we can open our favourite browser and paste the local network http://localhost:8080/
and we can perform full text search over the engine easily.

- Sample screenshot (Searching over 633K documents in 0.18 milliseconds!)

![screenshot](https://github.com/xkmsoft/wikisearcher/images/frontend.png)
Format: ![screenshot](https://github.com/xkmsoft/wikisearcher/images/frontend.png)