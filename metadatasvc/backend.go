package metadatasvc

type Backend interface {
	Sign()
	Verify()
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
