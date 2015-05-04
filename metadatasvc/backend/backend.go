package metadatasvc

type Backend interface {
	Sign()
	Verify()
	RegisterPod(uuid string, hmacKey string) error
	UnregisterPod(uuid string) error
}

type backend struct {
}

func NewBackend() Backend {
	var b Backend = &backend{}
	return b
}

func (b *backend) Sign() {

}

func (b *backend) Verify() {

}

func (b *backend) RegisterPod(uuid string, hmacKey string) error {
	return nil
}

func (b *backend) UnregisterPod(uuid string) error {
	return nil
}
