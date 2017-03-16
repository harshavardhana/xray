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
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	router "github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/minio/go-cv"
)

type xrayHandlers struct {
	sync.RWMutex

	// Used for calculating difference.
	prevSR sensorRecord

	// Represents client response channel, sends client data.
	clntRespCh chan interface{}

	// Display memory channels.
	displayCh, displayRecvCh chan bool

	// Used for upgrading the incoming HTTP
	// wconnection into a websocket wconnection.
	upgrader websocket.Upgrader
}

// Detects face objects on incoming data.
func (v *xrayHandlers) detectObjects(data []byte) {
	defer func() {
		if r := recover(); r != nil {
			errorIf(r.(error), "Recovered from a panic in detectObjects")
		}
	}()

	img, err := gocv.DecodeImageMem(data)
	if err != nil {
		errorIf(err, "Unable to decode incoming image")
		v.displayCh <- false
		fo := faceObject{
			Type:    Unknown,
			Display: <-v.displayRecvCh,
			Zoom:    -1,
		}
		v.clntRespCh <- fo
		return
	}

	// Lookup text.. in the picture and extract them.
	text := v.lookupText(img)
	if text != "" {
		log.Println("Detected textual data", text)
	}

	// Detect faces and decode their positions.
	faces := v.lookupFaces(img)

	// Array > 0 means we have detected faces.
	facesDetected := len(faces) > 0
	if !facesDetected {
		v.displayCh <- false
		fo := faceObject{
			Type:    Unknown,
			Display: <-v.displayRecvCh,
			Zoom:    -1,
		}
		v.clntRespCh <- fo
		return
	}

	v.displayCh <- facesDetected
	fo := faceObject{
		Type:      Human,
		Positions: faces,
		Display:   <-v.displayRecvCh,
		Zoom:      calculateOptimalZoomFactor(faces, img.Rect),
	}

	// Send the data to client.
	v.clntRespCh <- fo
}

// Detects motion based on sensor difference.
func (v *xrayHandlers) detectMotion(data []byte) {
	defer func() {
		if r := recover(); r != nil {
			errorIf(r.(error), "Recovered from a panic in detectMotion")
		}
	}()

	var sr sensorRecord
	if err := json.Unmarshal(data, &sr); err != nil {
		v.displayCh <- false
		fo := faceObject{
			Type:    Unknown,
			Display: <-v.displayRecvCh,
			Zoom:    0,
		}
		v.clntRespCh <- fo
		errorIf(err, "Unable to extract sensor record %s", string(data))
		return
	}

	v.displayCh <- v.shouldDisplayCamera(sr)
	fo := faceObject{
		Type:    Unknown,
		Display: <-v.displayRecvCh,
		Zoom:    0,
	}

	v.clntRespCh <- fo

	// Save current sendor info, needed to wake up camera.
	v.persistCurrentSensorR(sr)
}

// Detect detects metadata about the incoming data.
func (v *xrayHandlers) Detect(w http.ResponseWriter, r *http.Request) {
	wconn, err := v.upgrader.Upgrade(w, r, nil)
	if err != nil {
		errorIf(err, "Unable to perform websocket upgrade the request.")
		return
	}

	wc := wConn{wconn}
	defer wc.Close()

	// Waiting on incoming reads.
	for {
		mt, data, err := wc.ReadMessage()
		if err != nil {
			errorIf(err, "Unable to read incoming binary message.")
			break
		}

		if len(data) == 0 {
			continue
		}

		// Support if client sent a text message, most
		// probably its sensor or location metadata.
		if mt == websocket.TextMessage {
			// Ignore all other forms of incoming data.
			if !bytes.Contains(data, []byte("sensorName")) {
				continue
			}
			go v.detectMotion(data)
		} else if mt == websocket.BinaryMessage {
			go v.detectObjects(data)
		}
		wc.WriteMessage(mt, v.clntRespCh)
	}
}

// Initialize a new xray handlers.
func newXRayHandlers() *xrayHandlers {
	displayCh := make(chan bool)

	return &xrayHandlers{
		displayCh:     displayCh,
		displayRecvCh: displayMemoryRoutine(displayCh),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}, // use default options
	}
}

// Configure xray handler.
func configureXrayHandler(mux *router.Router) http.Handler {
	// Register all xray handlers.
	registerXRayRouter(mux)

	// Register additional routers if any.
	return mux
}

// Register xray router.
func registerXRayRouter(mux *router.Router) {

	// Initialize xray handlers.
	xray := newXRayHandlers()

	// xray Router
	xrayRouter := mux.NewRoute().PathPrefix("/").Subrouter()

	// Currently there is only one handler.
	xrayRouter.Methods("GET").HandlerFunc(xray.Detect)
}
