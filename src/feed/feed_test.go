package feed

import (
	"testing"

	// "github.com/schollz/kiki/src/logging"

	"github.com/stretchr/testify/assert"
)

var f *Feed

// BenchmarkGetUser-4         	     100	  17057282 ns/op
// BenchmarkShowFeed-4        	      20	  91682205 ns/op
// BenchmarkGetBasicPosts-4   	    2000	    666736 ns/op

func init() {
	var err error
	f, err = New("testdb", "4NfD9kWESGycUdbhbrFygNDjFun6NPk6utpkviyE1Ai6", "btbsjnjTtgi3aL9z2X8bqb1URVnCo3zqg4fC4co2JEu", false)
	if err != nil {
		panic(err)
	}
}

func TestShowFeed(t *testing.T) {
	f.Debug(true)
	_, err := f.ShowFeed(ShowFeedParameters{})
	assert.Nil(t, err)
	f.Debug(false)
}

func TestGetUser(t *testing.T) {
	u := f.GetUser()
	assert.Equal(t, "8cjeQPadXXCTGe9WbqER44CqduSHpqepX4tgAoEEFH4w", u.PublicKey)
	u = f.GetUser()
	assert.Equal(t, "8cjeQPadXXCTGe9WbqER44CqduSHpqepX4tgAoEEFH4w", u.PublicKey)
}

func BenchmarkGetUser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f.GetUser()
	}
}

func BenchmarkShowFeed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f.ShowFeed(ShowFeedParameters{})
	}
}

func BenchmarkGetBasicPosts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f.db.GetBasicPosts()
	}
}
