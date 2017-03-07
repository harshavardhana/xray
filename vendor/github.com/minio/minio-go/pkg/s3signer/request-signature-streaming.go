/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2017 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package s3signer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Reference for constants used below -
// http://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html#example-signature-calculations-streaming
const (
	streamingSignAlgorithm = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	streamingEncoding      = "aws-chunked"
	streamingPayloadHdr    = "AWS4-HMAC-SHA256-PAYLOAD"
	emptySHA256            = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	payloadChunkSize       = 64 * 1024
	chunkSigConstLen       = 17 // ";chunk-signature="
	signatureStrLen        = 64 // e.g. "f2ca1bb6c7e907d06dafe4687e579fce76b37e4e93b7605022da52e6ccc26fd2"
	crlfLen                = 2  // CRLF
)

// Request headers to be ignored while calculating seed signature for
// a request.
var ignoredStreamingHeaders = map[string]bool{
	"Authorization": true,
	"User-Agent":    true,
	"Content-Type":  true,
}

// getSignedChunkLength - calculates the length of chunk metadata
func getSignedChunkLength(chunkDataSize int64) int64 {
	return int64(len(fmt.Sprintf("%x", chunkDataSize))) +
		chunkSigConstLen +
		signatureStrLen +
		crlfLen +
		chunkDataSize +
		crlfLen
}

// getStreamLength - calculates the length of the overall stream (data + metadata)
func getStreamLength(dataLen, chunkSize int64) int64 {
	if dataLen <= 0 {
		return 0
	}
	chunksCount := int64(dataLen / chunkSize)
	remainingBytes := int64(dataLen % chunkSize)
	streamLen := int64(0)
	streamLen += chunksCount * getSignedChunkLength(chunkSize)
	if remainingBytes > 0 {
		streamLen += getSignedChunkLength(remainingBytes)
	}
	streamLen += getSignedChunkLength(0)
	return streamLen
}

// getChunkStringToSign - returns the string to sign given chunk data
// and previous signature.
func getChunkStringToSign(t time.Time, region, previousSig string, chunkData []byte) string {
	stringToSignParts := []string{
		streamingPayloadHdr,
		t.Format(iso8601DateFormat),
		getScope(region, t),
		previousSig,
		emptySHA256,
		hex.EncodeToString(sum256(chunkData)),
	}

	return strings.Join(stringToSignParts, "\n")
}

// prepareStreamingRequest - prepares a request with appropriate
// headers before computing the seed signature.
func prepareStreamingRequest(req *http.Request, dataLen int64, timestamp time.Time) {
	// Set x-amz-content-sha256 header.
	req.Header.Set("X-Amz-Content-Sha256", streamingSignAlgorithm)
	req.Header.Set("Content-Encoding", streamingEncoding)
	req.Header.Set("X-Amz-Date", timestamp.Format(iso8601DateFormat))
	// Set content length with streaming signature for each chunk included.
	req.Header.Set("x-amz-decoded-content-length", strconv.FormatInt(dataLen, 10))
}

// getChunkHeader - returns the chunk header.
// e.g string(IntHexBase(chunk-size)) + ";chunk-signature=" + signature + \r\n + chunk-data + \r\n
func getChunkHeader(chunkLen int64, signature string) []byte {
	return []byte(strconv.FormatInt(chunkLen, 16) + ";chunk-signature=" + signature + "\r\n")
}

// getChunkSignature - returns chunk signature for a given chunk and previous signature.
func getChunkSignature(chunkData []byte, reqTime time.Time, region, previousSignature, secretAccessKey string) string {
	chunkStringToSign := getChunkStringToSign(reqTime, region, previousSignature, chunkData)
	signingKey := getSigningKey(secretAccessKey, region, reqTime)
	return getSignature(signingKey, chunkStringToSign)
}

// getSeedSignature - returns the seed signature for a given request.
func (s *StreamingReader) setSeedSignature(req *http.Request) {
	// Get canonical request
	canonicalRequest := getCanonicalRequest(*req, ignoredStreamingHeaders)

	// Get string to sign from canonical request.
	stringToSign := getStringToSignV4(s.reqTime, s.region, canonicalRequest)

	signingKey := getSigningKey(s.secretAccessKey, s.region, s.reqTime)

	// Calculate signature.
	s.seedSignature = getSignature(signingKey, stringToSign)
}

// StreamingReader implements chunked upload signature as a reader on
// top of req.Body's ReaderCloser chunk header;data;... repeat
type StreamingReader struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	prevSignature   string
	seedSignature   string
	contentLen      int64
	baseReadCloser  io.ReadCloser
	bytesRead       int64
	buf             bytes.Buffer
	chunkBuf        []byte
	done            bool
	reqTime         time.Time
}

// signChunk - signs a chunk read from s.baseReader of chunkLen size.
func (s *StreamingReader) signChunk(chunkLen int) error {
	// Compute chunk signature for next header.
	signature := getChunkSignature(s.chunkBuf[:chunkLen], s.reqTime, s.region,
		s.prevSignature, s.secretAccessKey)

	// For next chunk signature computation.
	s.prevSignature = signature

	// Write chunk header into streaming buffer.
	chunkHdr := getChunkHeader(int64(chunkLen), signature)
	_, err := s.buf.Write(chunkHdr)
	if err != nil {
		return err
	}

	// Write chunk data into streaming buffer.
	s.buf.Write(s.chunkBuf[:chunkLen])
	s.buf.Write([]byte("\r\n"))
	return nil
}

// setStreamingAuthHeader - builds and sets authorization header value
// for streaming signature.
func (s *StreamingReader) setStreamingAuthHeader(req *http.Request) {
	credential := GetCredential(s.accessKeyID, s.region, s.reqTime)
	authParts := []string{
		signV4Algorithm + " Credential=" + credential,
		"SignedHeaders=" + getSignedHeaders(*req, ignoredStreamingHeaders),
		"Signature=" + s.seedSignature,
	}

	// Set authorization header.
	auth := strings.Join(authParts, ",")
	req.Header.Set("Authorization", auth)
}

// StreamingSignV4 - provides chunked upload signatureV4 support by
// implementing io.Reader.
func StreamingSignV4(req *http.Request, accessKeyID, secretAccessKey, region string, dataLen int64) {
	reqTime := time.Now().UTC()

	stReader := &StreamingReader{
		baseReadCloser:  req.Body,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		region:          region,
		reqTime:         reqTime,
		chunkBuf:        make([]byte, payloadChunkSize),
	}

	streamContentLen := getStreamLength(dataLen, int64(payloadChunkSize))
	req.ContentLength = streamContentLen

	// Add the request headers required for chunk upload signing.
	prepareStreamingRequest(req, dataLen, reqTime)
	stReader.contentLen = dataLen

	// Compute the seed signature.
	stReader.setSeedSignature(req)

	// Set the authorization header with the seed signature.
	stReader.setStreamingAuthHeader(req)

	// Set seed signature as prevSignature for subsequent
	// streaming signing process.
	stReader.prevSignature = stReader.seedSignature
	req.Body = stReader
}

// Read - this method performs chunk upload signature providing a
// io.Reader interface.
func (s *StreamingReader) Read(buf []byte) (int, error) {
	switch {
	// After the last chunk is read from underlying reader, we
	// never re-fill s.buf.
	case s.done:

	// s.buf will be (re-)filled with next chunk when has lesser
	// bytes than asked for.
	case s.buf.Len() < len(buf):
		n1, err := s.baseReadCloser.Read(s.chunkBuf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return 0, err
		}

		// Add bytesRead in case of a short read from
		// underlying reader.
		if err == io.ErrUnexpectedEOF {
			s.bytesRead += int64(n1)
		}

		// No more data left in baseReader - last chunk.
		if err == io.EOF {
			// bytes read from baseReader different than
			// content length provided.
			if s.bytesRead != s.contentLen {
				return 0, io.ErrUnexpectedEOF
			}

			// Done reading the last chunk from baseReader.
			s.done = true

			// Sign the chunk and write it to s.buf.
			err = s.signChunk(0)
			if err != nil {
				return 0, err
			}

		} else {
			// Re-slice to bytes read.
			s.chunkBuf = s.chunkBuf[:n1]
			s.bytesRead += int64(n1)

			// Sign the chunk and write it to s.buf.
			s.signChunk(n1)
		}

	default:
		// Just read out of s.buf.

	}
	return s.buf.Read(buf)
}

// Close - this method makes underlying io.ReadCloser's Close method available.
func (s *StreamingReader) Close() error {
	return s.baseReadCloser.Close()
}