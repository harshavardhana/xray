// +build ignore

/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package main

import (
	"log"
	"time"

	"github.com/minio/minio-go"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-bucketname and my-objectname
	// are dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("play.minio.io:9000", "Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", true)
	if err != nil {
		log.Fatalln(err)
	}

	presignedURL, err := s3Client.PresignedPutObject("images", "IMG-20160726-WA0002.jpg", time.Duration(1000)*time.Second)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(presignedURL)
}