package codecommit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	nurl "net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	keyPrefix        = "AWS4"
	authHeaderPrefix = "AWS4-HMAC-SHA256"
	requestType      = "aws4_request"
	timeFormat       = "20060102T150405"
	shortTimeFormat  = "20060102"
)

type SigningCtx struct {
	ServiceName string
	Region      string
	Time        time.Time
	CredValues  credentials.Value
	url         *nurl.URL

	formattedTime      string
	formattedShortTime string
	authHeaderPrefix   string
	requestType        string
}

func (ctx *SigningCtx) buildTime() {
	ctx.formattedTime = ctx.Time.UTC().Format(timeFormat)
	ctx.formattedShortTime = ctx.Time.UTC().Format(shortTimeFormat)
}

func (ctx *SigningCtx) getCanonicalString() string {
	method := "GIT"
	canonicalString := fmt.Sprintf("%s\n%s\n\nhost:%s\n\nhost\n", method, ctx.url.Path, ctx.url.Hostname())
	return strings.Join([]string{
		ctx.authHeaderPrefix,
		ctx.formattedTime,
		strings.Join([]string{
			ctx.formattedShortTime,
			ctx.Region,
			ctx.ServiceName,
			ctx.requestType,
		}, "/"),
		fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalString))),
	}, "\n")
}

func (ctx *SigningCtx) signCodeCommitRequest() string {
	_, hMAC := hMACSHA256([]byte(keyPrefix+ctx.CredValues.SecretAccessKey), []byte(ctx.formattedShortTime))
	params := []string{ctx.Region, ctx.ServiceName, ctx.requestType}
	for _, param := range params {
		hMAC([]byte(param))
	}
	sig := hex.EncodeToString(hMAC([]byte(ctx.getCanonicalString())))

	return fmt.Sprintf("%sZ%s", ctx.formattedTime, sig)
}

//
func NewSigningContext(u *nurl.URL, region, serviceName string, credValues credentials.Value, signTime time.Time) SigningCtx {
	ctx := &SigningCtx{
		url:         u,
		Region:      region,
		ServiceName: serviceName,
		CredValues:  credValues,
		Time:        signTime,

		authHeaderPrefix: authHeaderPrefix,
		requestType:      requestType,
	}
	ctx.buildTime()
	return *ctx
}

//
//
// Closure function for computing an HMAC-SH256. Each call to the closure uses
// the previous call's value as the key and returns the HMAC-SHA256 checksum
// Example:
//	hm := hMACSHA256([]byte("initial_key", []byte("initial_data"))
//	result := hm([]byte("more_data"))
//	result = hm([]byte("even_more_data"))
//	...
func hMACSHA256(key, data []byte) ([]byte, func(data []byte) []byte) {
	last := hMACSum256(key, data)

	return last, func(data []byte) []byte {
		last = hMACSum256(last, data)
		return last
	}
}

// Return the HMAC-SHA256 checksum for key and data
func hMACSum256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
