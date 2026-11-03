// Package webpush sends Web Push messages: payload encryption by RFC 8291
// (aes128gcm, RFC 8188) and application server identification by RFC 8292
// (VAPID). It uses only the standard library: crypto/ecdh for the key
// agreement, crypto/hkdf for the key schedule and crypto/ecdsa for the VAPID
// signature (ADR 0018).
package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// RecordSize is the aes128gcm record size used. A payload is sent as one
	// record, so it must fit within it (RFC 8291 section 4).
	RecordSize = 4096
	// MaxPayload is the largest plaintext accepted: one record, less the
	// header (86 bytes), the padding delimiter and the AEAD tag.
	MaxPayload = RecordSize - 86 - 1 - 16
	// jwtLifetime is how long a VAPID token is valid; RFC 8292 allows up to 24 hours.
	jwtLifetime = 12 * time.Hour
	// jwtReuse is how long a signed token is reused for one push service.
	jwtReuse = time.Hour
)

// ErrPayloadTooLarge is returned for a payload that does not fit one record.
var ErrPayloadTooLarge = errors.New("push payload is too large")

var b64 = base64.RawURLEncoding

// decodeKey reads base64url with or without padding, and also standard
// base64, as browsers and libraries have each been seen to send.
func decodeKey(value string) ([]byte, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "=")
	if decoded, err := b64.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

// SubscriptionKeys checks a subscription's keys: p256dh must be an
// uncompressed point on P-256 and auth a 16-byte secret.
func SubscriptionKeys(p256dh, auth string) (*ecdh.PublicKey, []byte, error) {
	raw, err := decodeKey(p256dh)
	if err != nil {
		return nil, nil, fmt.Errorf("p256dh is not base64url: %w", err)
	}
	public, err := ecdh.P256().NewPublicKey(raw)
	if err != nil || len(raw) != 65 {
		return nil, nil, errors.New("p256dh is not a P-256 public key")
	}
	secret, err := decodeKey(auth)
	if err != nil {
		return nil, nil, fmt.Errorf("auth is not base64url: %w", err)
	}
	if len(secret) != 16 {
		return nil, nil, errors.New("auth must be 16 bytes")
	}
	return public, secret, nil
}

// Encrypt encrypts a payload for a subscription (RFC 8291) as one aes128gcm
// record with a fresh application server key pair and salt.
func Encrypt(plaintext []byte, p256dh, auth string) ([]byte, error) {
	public, secret, err := SubscriptionKeys(p256dh, auth)
	if err != nil {
		return nil, err
	}
	ephemeral, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return encrypt(plaintext, public, secret, ephemeral, salt)
}

// encrypt is Encrypt with the key pair and salt given, for RFC 8291's test vector.
func encrypt(plaintext []byte, receiver *ecdh.PublicKey, authSecret []byte, sender *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	if len(plaintext) > MaxPayload {
		return nil, ErrPayloadTooLarge
	}
	shared, err := sender.ECDH(receiver)
	if err != nil {
		return nil, err
	}
	receiverBytes, senderBytes := receiver.Bytes(), sender.PublicKey().Bytes()
	keyInfo := "WebPush: info\x00" + string(receiverBytes) + string(senderBytes)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, keyInfo, 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// The last (only) record ends with the delimiter 0x02 and no padding.
	record := append(append([]byte{}, plaintext...), 0x02)

	var body bytes.Buffer
	body.Write(salt)
	_ = binary.Write(&body, binary.BigEndian, uint32(RecordSize))
	body.WriteByte(byte(len(senderBytes)))
	body.Write(senderBytes)
	body.Write(gcm.Seal(nil, nonce, record, nil))
	return body.Bytes(), nil
}

// VAPID is the application server's identity (RFC 8292).
type VAPID struct {
	// PublicKey is the uncompressed P-256 point, base64url: the
	// applicationServerKey browsers subscribe with.
	PublicKey string
	// Subject is a mailto: or https: contact for push services.
	Subject string

	private *ecdsa.PrivateKey
	now     func() time.Time

	mu     sync.Mutex
	tokens map[string]token
}

type token struct {
	value   string
	expires time.Time
}

// GenerateKeys returns a new VAPID key pair, base64url without padding: the
// public key as an uncompressed point and the private key as its scalar.
func GenerateKeys() (publicKey, privateKey string, err error) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	raw, err := private.Bytes()
	if err != nil {
		return "", "", err
	}
	public, err := private.PublicKey.Bytes()
	if err != nil {
		return "", "", err
	}
	return b64.EncodeToString(public), b64.EncodeToString(raw), nil
}

// NewVAPID checks a key pair and subject and returns the identity. The
// public key must be the private key's own.
func NewVAPID(publicKey, privateKey, subject string) (*VAPID, error) {
	subject = strings.TrimSpace(subject)
	if !strings.HasPrefix(subject, "mailto:") && !strings.HasPrefix(subject, "https://") {
		return nil, errors.New("VAPID subject must be a mailto: or https: URL")
	}
	rawPrivate, err := decodeKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("VAPID private key is not base64url: %w", err)
	}
	private, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), rawPrivate)
	if err != nil {
		return nil, fmt.Errorf("VAPID private key: %w", err)
	}
	rawPublic, err := decodeKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("VAPID public key is not base64url: %w", err)
	}
	derived, err := private.PublicKey.Bytes()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(derived, rawPublic) {
		return nil, errors.New("VAPID public key does not belong to the private key")
	}
	return &VAPID{PublicKey: b64.EncodeToString(rawPublic), Subject: subject, private: private, tokens: map[string]token{}}, nil
}

// Authorization returns the Authorization header for a push endpoint:
// "vapid t=<JWT>, k=<public key>". Tokens are reused per push service for an
// hour.
func (vapid *VAPID) Authorization(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("endpoint is not an absolute URL")
	}
	audience := parsed.Scheme + "://" + parsed.Host
	now := time.Now()
	if vapid.now != nil {
		now = vapid.now()
	}
	vapid.mu.Lock()
	defer vapid.mu.Unlock()
	if cached, ok := vapid.tokens[audience]; ok && now.Before(cached.expires) {
		return "vapid t=" + cached.value + ", k=" + vapid.PublicKey, nil
	}
	signed, err := vapid.sign(audience, now.Add(jwtLifetime))
	if err != nil {
		return "", err
	}
	if vapid.tokens == nil {
		vapid.tokens = map[string]token{}
	}
	vapid.tokens[audience] = token{value: signed, expires: now.Add(jwtReuse)}
	return "vapid t=" + signed + ", k=" + vapid.PublicKey, nil
}

// sign makes an ES256 JWT with the claims RFC 8292 requires.
func (vapid *VAPID) sign(audience string, expires time.Time) (string, error) {
	header := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, err := json.Marshal(map[string]any{"aud": audience, "exp": expires.Unix(), "sub": vapid.Subject})
	if err != nil {
		return "", err
	}
	input := header + "." + b64.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, vapid.private, digest[:])
	if err != nil {
		return "", err
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return input + "." + b64.EncodeToString(signature), nil
}

// Urgency values (RFC 8030 section 5.3).
const (
	UrgencyVeryLow = "very-low"
	UrgencyLow     = "low"
	UrgencyNormal  = "normal"
	UrgencyHigh    = "high"
)

// topicPattern is RFC 8030's Topic: up to 32 characters of the URL-safe
// base64 alphabet.
var topicPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// TopicFor turns a notice's tag into a Topic header value, so a newer notice
// with the same tag replaces an undelivered older one at the push service.
func TopicFor(tag string) string {
	sum := sha256.Sum256([]byte(tag))
	return b64.EncodeToString(sum[:])[:32]
}

// Message is one push message to one subscription.
type Message struct {
	Endpoint string
	P256dh   string
	Auth     string
	Payload  []byte
	// TTL is how long the push service may hold the message for a device
	// that is offline; 0 means deliver now or never.
	TTL     time.Duration
	Urgency string
	Topic   string
}

// Result is a push service's answer.
type Result struct {
	Status int
	// Gone means the subscription no longer exists (404 or 410) and should be deleted.
	Gone bool
	// RetryAfter is the delay a 429 or 503 asked for, if it gave one.
	RetryAfter time.Duration
}

// Accepted reports whether the push service took the message.
func (result Result) Accepted() bool { return result.Status >= 200 && result.Status < 300 }

// Client sends messages.
type Client struct {
	VAPID *VAPID
	HTTP  *http.Client
}

// Send encrypts and posts one message. An error is returned only when the
// request could not be made or answered; a push service's refusal is in the
// Result.
func (client *Client) Send(ctx context.Context, message Message) (Result, error) {
	if message.Topic != "" && !topicPattern.MatchString(message.Topic) {
		return Result{}, errors.New("topic must be at most 32 URL-safe base64 characters")
	}
	body, err := Encrypt(message.Payload, message.P256dh, message.Auth)
	if err != nil {
		return Result{}, err
	}
	authorization, err := client.VAPID.Authorization(message.Endpoint)
	if err != nil {
		return Result{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, message.Endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Content-Encoding", "aes128gcm")
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("TTL", strconv.Itoa(max(0, int(message.TTL/time.Second))))
	if message.Urgency != "" {
		request.Header.Set("Urgency", message.Urgency)
	}
	if message.Topic != "" {
		request.Header.Set("Topic", message.Topic)
	}
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	result := Result{Status: response.StatusCode}
	switch response.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		result.Gone = true
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		result.RetryAfter = retryAfter(response.Header.Get("Retry-After"), time.Now())
	}
	return result, nil
}

// retryAfter reads Retry-After as seconds or an HTTP date.
func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}
