package cmd

//#cgo LDFLAGS: -llept -ltesseract
//#include "tess.h"

import "C"
import "unsafe"

// Tess executes tesseract only with source image buffer.
func Tess(imgBuf []byte, languages string) string {
	buf := unsafe.Pointer(&buf[0])
	len := C.CSize_t(len(buf))
	lang := C.CString(languages)
	return C.GoString(C.tess(buf, len, lang))
}

func (v xrayHandlers) lookupText(img []byte) string {
	// By default only look for english texts.
	return Tess(img, "eng")
}
