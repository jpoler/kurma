package backend

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/appc/spec/schema"
)

var (
	ErrInvalidUUID      = errors.New("invalid uuid")
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidSignature = errors.New("invalid HMAC signature")
)

type Backend interface {
	Sign(token, content string) (string, error)
	Verify(content, signature, uuid string) error
	RegisterPod(uuid string, manifest []byte, hmacKey string) (string, error)
	UnregisterPod(uuid string) error
	GetPod(token string) *schema.PodManifest
	GetPodUUID(token string) (string, error)
}

type podData struct {
	Manifest *schema.PodManifest
	HMACKey  string
	UUID     string
}

type backend struct {
	sync.RWMutex
	randGen      *rand.Rand
	tokensByUUID map[string]string
	podsByToken  map[string]*podData
}

// NewBackend ...
func NewBackend() Backend {
	var b Backend = &backend{
		tokensByUUID: make(map[string]string),
		podsByToken:  make(map[string]*podData),
		randGen:      rand.New(rand.NewSource(time.Now().UnixNano())),
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
	token := b.makeToken()

	if hmacKey == "" {
		hmacKey = b.makeToken()
	}

	b.podsByToken[token] = &podData{
		Manifest: podDef,
		HMACKey:  hmacKey,
		UUID:     uuid,
	}
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

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")

func (b *backend) makeToken() string {
	tokenRunes := make([]rune, 26) // ~155 bits of entropy
	for i := range tokenRunes {
		tokenRunes[i] = letters[b.randGen.Intn(len(letters))]
	}

	return string(tokenRunes)
}

// GetPod ..
func (b *backend) GetPod(token string) *schema.PodManifest {
	b.RLock()
	defer b.RUnlock()
	return b.getPod(token)
}

func (b *backend) getPod(token string) *schema.PodManifest {
	pod, ok := b.podsByToken[token]
	if !ok {
		return nil
	}

	return pod.Manifest
}

// GetPodUUID ..
func (b *backend) GetPodUUID(token string) (string, error) {
	b.RLock()
	defer b.RUnlock()
	return b.getPodUUID(token)
}

func (b *backend) getPodUUID(token string) (string, error) {
	pod, ok := b.podsByToken[token]
	if !ok {
		return "", ErrInvalidToken
	}

	return pod.UUID, nil
}

// Sign ...
func (b *backend) Sign(token, content string) (string, error) {
	b.RLock()
	defer b.RUnlock()
	return b.sign(token, content)
}

func (b *backend) sign(token, content string) (string, error) {
	pod, ok := b.podsByToken[token]
	if !ok {
		return "", ErrInvalidToken
	}

	hash := hmac.New(sha512.New, []byte(pod.HMACKey))
	hash.Write([]byte(content))
	out := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return out, nil
}

// Verify ...
func (b *backend) Verify(content, signature, uuid string) error {
	b.RLock()
	defer b.RUnlock()
	return b.verify(content, signature, uuid)
}

func (b *backend) verify(content, signature, uuid string) error {
	token, ok := b.tokensByUUID[uuid]
	if !ok {
		return ErrInvalidUUID
	}

	pod, ok := b.podsByToken[token]
	if !ok {
		return ErrInvalidToken
	}

	// Encode
	hmacEncoded := hmac.New(sha512.New, []byte(pod.HMACKey))
	hmacEncoded.Write([]byte(content))
	// Decode
	decodedSig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	if !hmac.Equal(decodedSig, hmacEncoded.Sum(nil)) {
		return ErrInvalidSignature
	}

	return nil
}
