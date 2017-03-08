/*
 * Copyright (c) 2017 Minio, Inc. <https://www.minio.io>
 *
 * This file is part of Xray.
 *
 * Xray is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package cmd

import (
	"os"
	"sync"
	"unsafe"

	"github.com/minio/go-cv"
	"github.com/minio/xray/cascade"
)

func init() {
	for i := 0; i < globalDetectParallel; i++ {
		if globalIsLBP {
			globalDetect[i] = gocv.DetectionLoadABuf(cascade.MustAsset("lbp_face.xml"))
		} else {
			globalDetect[i] = gocv.DetectionLoadABuf(cascade.MustAsset("haar_face_0.xml"))
		}
	}
}

// Global constants for Xray.
const (
	minGoVersion         = ">= 1.7" // Xray requires at least Go v1.7
	globalDetectParallel = 8
)

var (
	globalXrayCertFile = "/etc/ssl/public.crt"
	globalXrayKeyFile  = "/etc/ssl/private.key"

	globalDebug = os.Getenv("DEBUG") != ""
	globalIsLBP = os.Getenv("LBP_CASCADE") != ""

	globalDetectMutex [globalDetectParallel]sync.Mutex
	globalDetect      [globalDetectParallel]unsafe.Pointer

	globalHaarCascadeFile = "haar_face_0.xml"
	globalLBPCascadeFile  = "lbp_face.xml"
	// List of all other classifiers.
)
