package main

import "github.com/apcera/kurma/metadatasvc"

func main() {
	mds := metadatasvc.NewMetadataService()
	mds.Listen()
}
