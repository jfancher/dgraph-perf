package main

import (
	"math/rand"
	"strings"
)

var charPool = []rune("abcdefghijklmnopqrstuvwxyz1234567890 ")

func randomString(r *rand.Rand, n int) string {
	var buf strings.Builder
	buf.Grow(n)
	for i := 0; i < n; i++ {
		buf.WriteRune(charPool[r.Intn(len(charPool))])
	}
	return buf.String()
}
