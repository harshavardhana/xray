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

import "github.com/lazywei/go-opencv/opencv"

func getFacePositions(faces []*opencv.Rect) (facePositions []facePosition, faceFound bool) {
	for _, value := range faces {
		if value.X() == 0 || value.Y() == 0 {
			continue
		}
		facePositions = append(facePositions, facePosition{
			PT1: opencv.Point{
				X: value.X() + value.Width(),
				Y: value.Y(),
			},
			PT2: opencv.Point{
				X: value.X(),
				Y: value.Y() + value.Height(),
			},
			Scalar:    255.0,
			Thickness: 3, // Border thickness defaulted to '3'.
			LineType:  1,
			Shift:     0,
		})
	}
	return facePositions, len(facePositions) > 0
}

// Saves current frame as previous frame for motion detection.
func (v *xrayHandlers) persistCurrFrame(currFrame *opencv.IplImage) {
	v.prevFrame = currFrame.Clone()
}

// Detects if one should display camera.
func (v *xrayHandlers) shouldDisplayCamera(currFrame *opencv.IplImage) {
	if !v.prevDisplay {
		if v.prevFrame != nil && currFrame != nil {
			v.prevDisplay = detectMovingFrames(v.prevFrame, currFrame)
			v.prevFrame.Release()
		}
	}
	v.metadataCh <- faceObject{
		Type:    Unknown,
		Display: v.prevDisplay,
		// Zoom level is zero if we don't detect any face.
	}
}

func (v *xrayHandlers) findFaces(currFrame *opencv.IplImage) (faces []*opencv.Rect) {
	gray := opencv.CreateImage(currFrame.Width(), currFrame.Height(), opencv.IPL_DEPTH_8U, 1)
	opencv.CvtColor(currFrame, gray, opencv.CV_BGR2GRAY)
	defer gray.Release()
	return globalHaarCascadeClassifier.DetectObjects(gray)
}
