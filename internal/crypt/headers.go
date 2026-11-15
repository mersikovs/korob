package crypt

const (
	HeaderEncrypted           = "X-Encrypted"
	HeaderEncryptedValue      = "hpke-v1"
	HeaderInnerEncoding       = "X-Inner-Encoding"
	HeaderOriginalContentType = "X-Original-Content-Type"
	ContentTypeCiphertext     = "application/octet-stream"
	InnerEncodingGzip         = "gzip"
)
