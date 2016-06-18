package store

import (
	"math/rand"
	"sync"
	"time"
)

var mutex sync.Mutex
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func int63() int64 {
	mutex.Lock()
	v := src.Int63()
	mutex.Unlock()
	return v
}

func randString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func (ds *MgoStore) GetNewImageShortCode() string {
	randTitle := randString(12)
	if ds.ExistsImageShortCode(randTitle) {
		randTitle = randString(12)
	}
	return randTitle
}

func (ds *MgoStore) GetNewAlbumShortCode() string {
	randTitle := randString(12)
	if ds.ExistsAlbumShortCode(randTitle) {
		randTitle = randString(12)
	}
	return randTitle
}