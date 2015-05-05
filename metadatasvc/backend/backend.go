package backend

import (
	"errors"
	"math/rand"
	"sync"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
)

type Backend interface {
	Sign(content, token string) (string, error)
	Verify(content, signature, uuid string) error
	RegisterPod(uuid string, manifest []byte, hmacKey string) (string, error)
	UnregisterPod(uuid string) error
	GetAppDefinition(token string, appName string) *schema.RuntimeApp
}

type backend struct {
	sync.RWMutex
	tokensByUUID map[string]string
	podsByToken  map[string]*schema.PodManifest
}

// NewBackend ...
func NewBackend() Backend {
	var b Backend = &backend{
		tokensByUUID: make(map[string]string),
		podsByToken:  make(map[string]*schema.PodManifest),
	}
	return b
}

// RegisterPod ...
func (b *backend) RegisterPod(uuid string, manifest []byte, hmacKey string) (string, error) {
	b.Lock()
	defer b.Unlock()
	return b.registerPod(uuid, manifest, hmacKey)
}

func (b *backend) registerPod(uuid string, manifest []byte, hmacKey string) (string, error) {
	podDef := schema.BlankPodManifest()
	err := podDef.UnmarshalJSON(manifest)
	if err != nil {
		return "", err
	}
	token := makeToken()

	b.podsByToken[token] = podDef
	b.tokensByUUID[uuid] = token

	return token, nil
}

// UnregisterPod ...
func (b *backend) UnregisterPod(uuid string) error {
	b.Lock()
	defer b.Unlock()
	return b.unregisterPod(uuid)
}

func (b *backend) unregisterPod(uuid string) error {
	token, ok := b.tokensByUUID[uuid]
	if !ok {
		return errors.New("Invalid POD")
	}

	delete(b.tokensByUUID, uuid)
	delete(b.podsByToken, token)
	return nil
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func makeToken() string {
	b := make([]rune, 26) // ~155 bits of entropy
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

// App Stuff
func (b *backend) GetAppDefinition(token string, appName string) *schema.RuntimeApp {
	b.RLock()
	defer b.RUnlock()
	return b.getAppDefinition(token, appName)
}

func (b *backend) getAppDefinition(token string, appName string) *schema.RuntimeApp {
	pod, ok := b.podsByToken[token]
	if !ok {
		return nil
	}
	return pod.Apps.Get(types.ACName(appName))
}

// Identity  stuff

// Sign ...
func (b *backend) Sign(content, token string) (string, error) {
	b.RLock()
	defer b.RUnlock()
	return b.sign(content, token)
}

func (b *backend) sign(content, token string) (string, error) {
	return "", nil
}

// Verify ...
func (b *backend) Verify(content, signature, uuid string) error {
	b.RLock()
	defer b.RUnlock()
	return b.verify(content, signature, uuid)
}

func (b *backend) verify(content, signature, uuid string) error {
	return nil
}
