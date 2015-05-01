package main

import "github.com/apcera/kurma/metadatasvc"

func main() {
	backend := metadatasvc.NewStore()
	mds := metadatasvc.NewRestServer(backend)
	mds.Listen()
}
