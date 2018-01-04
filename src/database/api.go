package database

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/schollz/kiki/src/letter"
)

func AddEnvelope(e letter.Envelope) (err error) {
	db, err := Open()
	if err != nil {
		return
	}
	defer db.Close()
	return db.addEnvelope(e)
}

func Set(bucket, key string, value interface{}) (err error) {
	db, err := Open()
	if err != nil {
		return
	}
	defer db.Close()
	return db.Set(bucket, key, value)
}

// GetEnvelopeFromID returns a single envelope from its ID
func GetEnvelopeFromID(id string) (e letter.Envelope, err error) {
	db, err := Open()
	if err != nil {
		return
	}
	defer db.Close()
	var es []letter.Envelope
	es, err = db.getAllFromPreparedQuery("SELECT * FROM letters WHERE id = ?", id)
	if err != nil {
		err = errors.Wrap(err, "GetEnvelopeFromID("+id+")")
	} else {
		e = es[0]
	}
	return
}

// GetAllEnvelopes returns all envelopes determined by whether they are opened
func GetAllEnvelopes(opened bool) (e []letter.Envelope, err error) {
	db, err := Open()
	if err != nil {
		return
	}
	defer db.Close()
	if opened {
		return db.getAllFromQuery("SELECT * FROM letters WHERE opened == 1")
	} else {
		return db.getAllFromQuery("SELECT * FROM letters WHERE opened == 0")
	}
}
