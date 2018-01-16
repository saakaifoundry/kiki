package feed

import (
	"fmt"
	"testing"

	// "github.com/schollz/kiki/src/logging"

	"github.com/stretchr/testify/assert"
)

func TestGetUser(t *testing.T) {
	f, err := Open(".")
	assert.Nil(t, err)
	f.Debug(false)
	u := f.GetUser()
	assert.Equal(t, "5z_8ZHf6cnZnortmafG0gsSX0Dl5jaOdCHUNoQiI5h8=", u.PublicKey)

	fmt.Println("HI")
}

func BenchmarkGetUser(b *testing.B) {
	f, err := Open(".")
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		f.GetUser()
	}
}

func BenchmarkGetPost(b *testing.B) {
	f, err := Open(".")
	f.Debug(false)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		f.ShowFeed(ShowFeedParameters{})
	}
}