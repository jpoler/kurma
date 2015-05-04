package metadatasvc

import (
	"sync"

	backend "github.com/apcera/kurma/metadatasvc/backend"
	"github.com/apcera/kurma/metadatasvc/grpc"
	rest "github.com/apcera/kurma/metadatasvc/rest"
)

// MetadataService does stuff.
type MetadataService interface {
	Listen() error
}

type metadataService struct {
	backend backend.Backend
	grpc    grpc.GrpcServer
	rest    rest.RestServer
}

func NewMetadataService() MetadataService {
	store := backend.NewBackend()

	return &metadataService{
		backend: store,
		grpc:    grpc.NewGrpcServer(store),
		rest:    rest.NewRestServer(store),
	}
}

func (mds *metadataService) Listen() error {
	var wg sync.WaitGroup

	// TODO: error propagation channel
	wg.Add(1)
	go func() {
		mds.rest.Listen()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		mds.grpc.Listen()
		wg.Done()
	}()

	wg.Wait()

	return nil
}
