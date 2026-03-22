//go:build linux && cgo

package cracker

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <crypt.h>
#include <string.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func verifyCandidatePassword(candidate string, hash string) (bool, error) {
	data := C.struct_crypt_data{}
	C.memset(unsafe.Pointer(&data), 0, C.size_t(unsafe.Sizeof(data)))

	cHash := C.CString(hash)
	cPass := C.CString(candidate)
	defer C.free(unsafe.Pointer(cHash))
	defer C.free(unsafe.Pointer(cPass))

	res := C.crypt_r(cPass, cHash, &data)
	if res == nil {
		return false, fmt.Errorf("crypt_r failed")
	}

	return C.GoString(res) == hash, nil
}
