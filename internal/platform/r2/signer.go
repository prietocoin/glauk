package r2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	algorithm   = "AWS4-HMAC-SHA256"
	serviceName = "s3"
	regionName  = "auto"
)

// signer implementa SigV4 sin SDK: R2 solo necesita PUT/GET y un subconjunto
// pequeño de cabeceras, así que no se arrastra toda la cadena de AWS.
type signer struct {
	accessKey string
	secretKey string
}

// sign añade x-amz-date, x-amz-content-sha256 y Authorization a la petición.
func (s signer) sign(req *http.Request, payload []byte, now time.Time) {
	stamp := now.UTC()
	amzDate := stamp.Format("20060102T150405Z")
	shortDate := stamp.Format("20060102")
	payloadHash := sha256Hex(payload)

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalHeaders, signedHeaders := canonicalHeaders(req, payloadHash)
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL),
		canonicalQuery(req.URL),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{shortDate, regionName, serviceName, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		algorithm, amzDate, scope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	key := signingKey(s.secretKey, shortDate)
	signature := hex.EncodeToString(hmacSHA256(key, []byte(stringToSign)))
	req.Header.Set("Authorization", algorithm+" Credential="+s.accessKey+
		"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func canonicalHeaders(req *http.Request, payloadHash string) (string, string) {
	values := map[string]string{
		"host":                 req.URL.Host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           req.Header.Get("x-amz-date"),
	}
	if contentType := strings.TrimSpace(req.Header.Get("Content-Type")); contentType != "" {
		values["content-type"] = contentType
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	var canonical strings.Builder
	for _, name := range names {
		canonical.WriteString(name)
		canonical.WriteString(":")
		canonical.WriteString(normalizeSpaces(values[name]))
		canonical.WriteString("\n")
	}
	return canonical.String(), strings.Join(names, ";")
}

func canonicalURI(target *url.URL) string {
	path := target.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}

func canonicalQuery(target *url.URL) string {
	query := target.Query()
	if len(query) == 0 {
		return ""
	}
	return query.Encode()
}

func normalizeSpaces(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func signingKey(secret, date string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	regionKey := hmacSHA256(dateKey, []byte(regionName))
	serviceKey := hmacSHA256(regionKey, []byte(serviceName))
	return hmacSHA256(serviceKey, []byte("aws4_request"))
}
