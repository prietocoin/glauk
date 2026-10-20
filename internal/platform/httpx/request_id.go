package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

func validRequestID(id string) bool {
	switch len(id) {
	case 36:
		return validUUID(id)
	case 26:
		return validULID(id)
	default:
		return false
	}
}

func validUUID(id string) bool {
	for i := 0; i < len(id); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if id[i] != '-' {
				return false
			}
			continue
		}
		if !isHex(id[i]) {
			return false
		}
	}
	return true
}

func validULID(id string) bool {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for i := 0; i < len(id); i++ {
		if !strings.ContainsRune(alphabet, byteToUpper(id[i])) {
			return false
		}
	}
	return true
}

func isHex(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

func byteToUpper(ch byte) rune {
	if ch >= 'a' && ch <= 'z' {
		return rune(ch - 'a' + 'A')
	}
	return rune(ch)
}
