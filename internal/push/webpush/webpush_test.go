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
	"encoding/binary"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mustDecode(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := decodeKey(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

// TestRFC8291Vector encrypts the example of RFC 8291 Appendix A with its keys
// and salt and must produce its body exactly.
func TestRFC8291Vector(t *testing.T) {
	sender, err := ecdh.P256().NewPrivateKey(mustDecode(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	if got := b64.EncodeToString(sender.PublicKey().Bytes()); got != "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8" {
		t.Fatalf("sender public key = %s", got)
	}
	receiver, _, err := SubscriptionKeys("BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4", "BTBZMqHH6r4Tts7J_aSIgg")
	if err != nil {
		t.Fatal(err)
	}
	body, err := encrypt([]byte("When I grow up, I want to be a watermelon"), receiver, mustDecode(t, "BTBZMqHH6r4Tts7J_aSIgg"),
		sender, mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlw"))
	if err != nil {
		t.Fatal(err)
	}
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if got := b64.EncodeToString(body); got != want {
		t.Fatalf("body =\n%s\nwant\n%s", got, want)
	}
}

// decrypt is the user agent's side of RFC 8291, to read what Encrypt wrote.
func decrypt(t *testing.T, body []byte, receiver *ecdh.PrivateKey, authSecret []byte) []byte {
	t.Helper()
	salt, size, idLength := body[:16], binary.BigEndian.Uint32(body[16:20]), int(body[20])
	if size != RecordSize {
		t.Fatalf("record size = %d", size)
	}
	senderPublic, err := ecdh.P256().NewPublicKey(body[21 : 21+idLength])
	if err != nil {
		t.Fatal(err)
	}
	shared, err := receiver.ECDH(senderPublic)
	if err != nil {
		t.Fatal(err)
	}
	info := "WebPush: info\x00" + string(receiver.PublicKey().Bytes()) + string(senderPublic.Bytes())
	ikm, _ := hkdf.Key(sha256.New, shared, authSecret, info, 32)
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, body[21+idLength:], nil)
	if err != nil {
		t.Fatal(err)
	}
	end := bytes.LastIndexByte(plain, 0x02)
	if end < 0 {
		t.Fatal("no padding delimiter")
	}
	return plain[:end]
}

type subscriber struct {
	private *ecdh.PrivateKey
	auth    []byte
	p256dh  string
	authKey string
}

func newSubscriber(t *testing.T) subscriber {
	t.Helper()
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	return subscriber{private: private, auth: auth, p256dh: b64.EncodeToString(private.PublicKey().Bytes()), authKey: b64.EncodeToString(auth)}
}

func TestEncryptRoundTrip(t *testing.T) {
	user := newSubscriber(t)
	payload := []byte(`{"title":"නායයෑම් මට්ටම 3 · රත්නපුර","body":"…"}`)
	body, err := Encrypt(payload, user.p256dh, user.authKey)
	if err != nil {
		t.Fatal(err)
	}
	if got := decrypt(t, body, user.private, user.auth); !bytes.Equal(got, payload) {
		t.Fatalf("decrypted %q", got)
	}
	if _, err := Encrypt(bytes.Repeat([]byte("x"), MaxPayload+1), user.p256dh, user.authKey); err != ErrPayloadTooLarge {
		t.Fatalf("oversized payload: %v", err)
	}
}

func TestSubscriptionKeysRejectsBadKeys(t *testing.T) {
	user := newSubscriber(t)
	cases := map[string][2]string{
		"short auth":     {user.p256dh, b64.EncodeToString([]byte("short"))},
		"not a point":    {b64.EncodeToString(bytes.Repeat([]byte{4}, 65)), user.authKey},
		"not base64":     {"!!!", user.authKey},
		"compressed key": {b64.EncodeToString(append([]byte{2}, make([]byte, 32)...)), user.authKey},
	}
	for name, keys := range cases {
		if _, _, err := SubscriptionKeys(keys[0], keys[1]); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func testVAPID(t *testing.T) *VAPID {
	t.Helper()
	public, private, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	vapid, err := NewVAPID(public, private, "mailto:alerts@example.org")
	if err != nil {
		t.Fatal(err)
	}
	return vapid
}

func TestNewVAPIDChecksKeysAndSubject(t *testing.T) {
	public, private, _ := GenerateKeys()
	other, _, _ := GenerateKeys()
	if _, err := NewVAPID(other, private, "mailto:a@example.org"); err == nil {
		t.Error("mismatched public key accepted")
	}
	if _, err := NewVAPID(public, private, "a@example.org"); err == nil {
		t.Error("bare address accepted as subject")
	}
	if _, err := NewVAPID(public, "nope", "https://example.org"); err == nil {
		t.Error("bad private key accepted")
	}
	if _, err := NewVAPID(public, private, "https://example.org/contact"); err != nil {
		t.Error(err)
	}
}

// verifyJWT checks an ES256 token against a public key and returns its claims.
func verifyJWT(t *testing.T, value, publicKey string) map[string]any {
	t.Helper()
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q", value)
	}
	raw := mustDecode(t, publicKey)
	public, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), raw)
	if err != nil {
		t.Fatal(err)
	}
	signature := mustDecode(t, parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(public, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
		t.Fatal("signature does not verify")
	}
	var header map[string]string
	_ = json.Unmarshal(mustDecode(t, parts[0]), &header)
	if header["alg"] != "ES256" || header["typ"] != "JWT" {
		t.Fatalf("header %v", header)
	}
	var claims map[string]any
	_ = json.Unmarshal(mustDecode(t, parts[1]), &claims)
	return claims
}

func TestSendSetsHeadersAndSignsForTheAudience(t *testing.T) {
	user := newSubscriber(t)
	vapid := testVAPID(t)
	var got *http.Request
	var body []byte
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client := &Client{VAPID: vapid, HTTP: server.Client()}
	payload := []byte(`{"title":"test"}`)
	result, err := client.Send(context.Background(), Message{Endpoint: server.URL + "/push/abc", P256dh: user.p256dh, Auth: user.authKey,
		Payload: payload, TTL: 4 * time.Hour, Urgency: UrgencyHigh, Topic: TopicFor("landslide:district:LK-91")})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted() || result.Gone {
		t.Fatalf("result %+v", result)
	}
	for header, want := range map[string]string{"TTL": "14400", "Urgency": "high", "Content-Encoding": "aes128gcm",
		"Content-Type": "application/octet-stream", "Topic": TopicFor("landslide:district:LK-91")} {
		if value := got.Header.Get(header); value != want {
			t.Errorf("%s = %q, want %q", header, value, want)
		}
	}
	authorization := got.Header.Get("Authorization")
	tokenPart, keyPart, ok := strings.Cut(strings.TrimPrefix(authorization, "vapid t="), ", k=")
	if !ok || keyPart != vapid.PublicKey {
		t.Fatalf("authorization %q", authorization)
	}
	claims := verifyJWT(t, tokenPart, vapid.PublicKey)
	if claims["aud"] != server.URL || claims["sub"] != "mailto:alerts@example.org" {
		t.Fatalf("claims %v", claims)
	}
	if exp := int64(claims["exp"].(float64)); exp > time.Now().Add(24*time.Hour).Unix() {
		t.Fatalf("exp too far ahead: %d", exp)
	}
	if plain := decrypt(t, body, user.private, user.auth); !bytes.Equal(plain, payload) {
		t.Fatalf("server received %q", plain)
	}
}

func TestSendReportsGoneAndRetryAfter(t *testing.T) {
	user := newSubscriber(t)
	status := http.StatusGone
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "120")
		}
		w.WriteHeader(status)
	}))
	defer server.Close()
	client := &Client{VAPID: testVAPID(t), HTTP: server.Client()}
	message := Message{Endpoint: server.URL, P256dh: user.p256dh, Auth: user.authKey, Payload: []byte("{}")}
	result, err := client.Send(context.Background(), message)
	if err != nil || !result.Gone || result.Accepted() {
		t.Fatalf("410: %+v %v", result, err)
	}
	status = http.StatusTooManyRequests
	result, err = client.Send(context.Background(), message)
	if err != nil || result.Gone || result.RetryAfter != 2*time.Minute {
		t.Fatalf("429: %+v %v", result, err)
	}
}

func TestTopicForIsAValidTopic(t *testing.T) {
	for _, tag := range []string{"x", "landslide:district:LK-91", strings.Repeat("long", 100)} {
		if topic := TopicFor(tag); !topicPattern.MatchString(topic) || len(topic) != 32 {
			t.Errorf("TopicFor(%q) = %q", tag, topic)
		}
	}
	if TopicFor("a") == TopicFor("b") {
		t.Error("topics collide")
	}
}

func TestRetryAfterReadsDates(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	if got := retryAfter(now.Add(90*time.Second).Format(http.TimeFormat), now); got != 90*time.Second {
		t.Fatalf("date: %v", got)
	}
	if got := retryAfter("soon", now); got != 0 {
		t.Fatalf("garbage: %v", got)
	}
}
