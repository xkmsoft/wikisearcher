# Wiki Searcher

## Overview

Wiki searcher is a sample full text search engine (reverse indexed) project written in Go which consist of the following packages.

- **engine** package has a (reversed indexed) indexer which is responsible to index Wiki XML documents by tokenizing the abstract and title of the Wiki document by performing filtering (lower-casing and removing stop words) and stemming phase into the tokens.
- **tcpserver** package has a basic tcp server that is responsible to download and decompress the [Wiki XML dumps](https://dumps.wikimedia.org/) and initialize the indexer with provided document for indexing phase. After these steps tcp server accepts tcp connections and returns the query results.
- **tcpclient** package has a basic tcp client which makes tcp connections to the tcp server for the queries.
- **apiserver** package has a basic REST api server to perform queries over the engine. 

## Basic usage
