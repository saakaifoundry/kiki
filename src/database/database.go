package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/schollz/kiki/src/keypair"
	"github.com/schollz/kiki/src/letter"
	"github.com/schollz/kiki/src/logging"
	"github.com/schollz/kiki/src/purpose"
	flock "github.com/theckman/go-flock"
)

var (
	log = logging.Log
)

type database struct {
	name     string
	db       *sql.DB
	fileLock *flock.Flock
}

// open will open the database for transactions by first aquiring a filelock.
func open(fileName string, readOnly ...bool) (d *database, err error) {
	d = new(database)
	// convert the name to base64 for file writing
	d.name = fileName

	// if read-only, make sure the database exists
	if _, err = os.Stat(d.name); err != nil && len(readOnly) > 0 && readOnly[0] {
		err = errors.New(fmt.Sprintf("database '%s' does not exist", d.name))
		return
	}

	// obtain a lock on the database
	d.fileLock = flock.NewFlock(d.name + ".lock")
	for {
		locked, err := d.fileLock.TryLock()
		if err == nil && locked {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// check if it is a new database
	newDatabase := false
	if _, err := os.Stat(d.name); os.IsNotExist(err) {
		newDatabase = true
	}

	// open sqlite3 database
	d.db, err = sql.Open("sqlite3", d.name)
	if err != nil {
		return
	}

	// create new database tables if needed
	if newDatabase {
		err = d.MakeTables()
		if err != nil {
			return
		}
	}

	return
}

// Close will close the database connection and remove the filelock.
func (d *database) Close() (err error) {
	// close filelock
	err = d.fileLock.Unlock()
	if err != nil {
		log.Error(err)
	} else {
		os.Remove(d.name + ".lock")
	}
	// close database
	err2 := d.db.Close()
	if err2 != nil {
		err = err2
		log.Error(err)
	}
	return
}

// MakeTables creates two tables, a `keystore` table:
//
// 	BUCKET_KEY (TEXT)	VALUE (TEXT)
//
// and also a `letters`:
func (d *database) MakeTables() (err error) {
	sqlStmt := `create table keystore (bucket_key text not null primary key, value text);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		return
	}
	sqlStmt = `create index keystore_idx on keystore(bucket_key);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		return
	}
	// The "letters" table contains all the envelopes (opened and unopened) and their respective inforamtion in the letters.
	sqlStmt = `create table letters (id text not null primary key, time TIMESTAMP, sender text, signature text, sealed_recipients text, sealed_letter text, opened integer, letter_purpose text, letter_to text, letter_content text, letter_replaces text, letter_replyto text, unique(id), UNIQUE(signature));`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables, letters")
		return
	}

	// The following tables are for organizing the letter data to make it more easily (and quickly) parsed. These tables are filled in when regenerating the feed.

	// The "persons" table fills with public information about the people on the network with how they relate to you (following/follower/blocking) and how they prsent themselves (profile, name, image). All this information is determined by reading letters, but as letters determine these properties dynamically and chronologically, this table will ensure that the latest version is determined.
	sqlStmt = `CREATE TABLE persons (id INTEGER PRIMARY KEY, public_key TEXT, name TEXT, profile TEXT, image TEXT, following INTEGER, follower INTEGER, blocking INTEGER);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables, persons")
		return
	}
	// // The "keypairs" table fills with all the keys provided for friends, as well as keys from friends. When encrypting for friends it will only use keys for friends. When encrypting for friends of friends it will use all the keys. For decrypting, it will try every keypair.
	// // This table is filled in dynamically, inserting each keypair found into the table.
	// sqlStmt = `CREATE TABLE keypairs (id INTEGER PRIMARY KEY, persons_id integer, time TIMESTAMP, keypair TEXT);`
	// _, err = d.db.Exec(sqlStmt)
	// if err != nil {
	// 	err = errors.Wrap(err, "MakeTables, keypairs")
	// 	return
	// }
	return
}

// Get will retrieve the value associated with a key.
func (d *database) Get(bucket, key string, v interface{}) (err error) {
	stmt, err := d.db.Prepare("select value from keystore where bucket_key = ?")
	if err != nil {
		return errors.Wrap(err, "problem preparing SQL")
	}
	defer stmt.Close()
	var result string
	err = stmt.QueryRow(bucket + "/" + key).Scan(&result)
	if err != nil {
		return errors.Wrap(err, "problem getting key")
	}

	err = json.Unmarshal([]byte(result), &v)
	if err != nil {
		return
	}
	return
}

// Set will set a value in the database, when using it like a keystore.
func (d *database) Set(bucket, key string, value interface{}) (err error) {
	var b []byte
	b, err = json.Marshal(value)
	if err != nil {
		return err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	stmt, err := tx.Prepare("insert or replace into keystore(bucket_key,value) values (?, ?)")
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	defer stmt.Close()

	_, err = stmt.Exec(bucket+"/"+key, string(b))
	if err != nil {
		return errors.Wrap(err, "Set")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "Set")
	}

	return
}

// addEnvelope will add or replace an envelope
func (d *database) addEnvelope(e letter.Envelope) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	var opened int
	// marshaled things
	var mSealedRecipients, mTo string
	if e.Opened {
		opened = 1
	} else {
		opened = 0
	}
	var b []byte

	b, err = json.Marshal(e.SealedRecipients)
	if err != nil {
		return errors.Wrap(err, "problem marshaling SealedRecipients")
	}
	mSealedRecipients = string(b)

	b, err = json.Marshal(e.Letter.To)
	if err != nil {
		return errors.Wrap(err, "problem marshaling To")
	}
	mTo = string(b)

	stmt, err := tx.Prepare("insert or replace into letters(id,time,sender,signature,sealed_recipients,sealed_letter,opened,letter_purpose,letter_to,letter_content,letter_replaces,letter_replyto) values(?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(e.ID, e.Timestamp, e.Sender.Public, e.Signature, mSealedRecipients, e.SealedLetter, opened, e.Letter.Purpose, mTo, e.Letter.Content, e.Letter.Replaces, e.Letter.ReplyTo)
	if err != nil {
		return
	}
	tx.Commit()
	return
}

func (d *database) getAllFromQuery(query string) (s []letter.Envelope, err error) {
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		err = errors.Wrap(err, "getAllFromQuery")
		return
	}
	defer rows.Close()

	// parse rows
	s, err = d.getRows(rows)
	if err != nil {
		err = errors.Wrap(err, query)
	}
	return
}

// getAllFromPreparedQuery
func (d *database) getAllFromPreparedQuery(query string, args ...interface{}) (s []letter.Envelope, err error) {
	// prepare statement
	stmt, err := d.db.Prepare(query)
	if err != nil {
		err = errors.Wrap(err, query)
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query(args...)
	if err != nil {
		err = errors.Wrap(err, query)
		return
	}
	defer rows.Close()
	s, err = d.getRows(rows)
	if err != nil {
		err = errors.Wrap(err, query)
	}
	return
}

func (d *database) getRows(rows *sql.Rows) (s []letter.Envelope, err error) {
	s = make([]letter.Envelope, 100000)
	sI := 0
	// loop through rows
	err = errors.New("no data available")
	for rows.Next() {
		var e letter.Envelope
		e.Letter = letter.Letter{}
		var opened int
		// marshaled things
		var mSender, mSealedRecipients, mTo string
		err = rows.Scan(&e.ID, &e.Timestamp, &mSender, &e.Signature, &mSealedRecipients, &e.SealedLetter, &opened, &e.Letter.Purpose, &mTo, &e.Letter.Content, &e.Letter.Replaces, &e.Letter.ReplyTo)
		e.Sender, err = keypair.FromPublic(mSender)
		json.Unmarshal([]byte(mSealedRecipients), &e.SealedRecipients)
		json.Unmarshal([]byte(mTo), &e.Letter.To)

		e.Opened = opened == 1
		if err != nil {
			err = errors.Wrap(err, "getRows")
			return
		}

		s[sI] = e
		sI++
		err = nil
	}
	s = s[:sI]

	if err != nil {
		return
	}
	err = rows.Err()
	if err != nil {
		err = errors.Wrap(err, "getRows")
	}
	return
}

// getKeys returns all the keys shared with you in the database, which can be queried by the sender
func (d *database) getKeys(sender ...string) (s []keypair.KeyPair, err error) {
	var query string
	if len(sender) > 0 {
		query = fmt.Sprintf("SELECT letter_content FROM letters WHERE opened == 1 AND letter_purpose == '%s' AND sender == '%s' ORDER BY time DESC;", purpose.ShareKey, sender[0])
	} else {
		query = fmt.Sprintf("SELECT letter_content FROM letters WHERE opened == 1 AND letter_purpose == '%s' ORDER BY time DESC;", purpose.ShareKey)
	}
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		err = errors.Wrap(err, "getKeys")
		return
	}
	defer rows.Close()

	// parse rows
	s = make([]keypair.KeyPair, 100000)
	sI := 0
	// loop through rows
	for rows.Next() {
		var mKeyPair string
		err = rows.Scan(&mKeyPair)
		if err != nil {
			err = errors.Wrap(err, "getKeys")
			return
		}

		var kp keypair.KeyPair
		err = json.Unmarshal([]byte(mKeyPair), &kp)
		if err != nil {
			return
		}
		s[sI] = kp
		sI++
	}
	s = s[:sI]
	err = rows.Err()
	if err != nil {
		err = errors.Wrap(err, "getKeys")
	}
	return
}

// getIDs returns all the envelope IDs
func (d *database) getIDs() (s []string, err error) {
	query := fmt.Sprintf("SELECT id FROM letters ORDER BY time DESC;")
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		err = errors.Wrap(err, "getIDs")
		return
	}
	defer rows.Close()

	// parse rows
	s = make([]string, 1000000)
	sI := 0
	// loop through rows
	for rows.Next() {
		var mID string
		err = rows.Scan(&mID)
		if err != nil {
			err = errors.Wrap(err, "getIDs")
			return
		}
		s[sI] = mID
		sI++
	}
	s = s[:sI]
	err = rows.Err()
	if err != nil {
		err = errors.Wrap(err, "getIDs")
	}
	return
}

// getName returns the name of a person
func (d *database) getName(person string) (name string, err error) {
	query := fmt.Sprintf("SELECT letter_content FROM letters WHERE opened == 1 AND letter_purpose == '%s' AND sender == '%s' ORDER BY time DESC;", purpose.AssignName, person)
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		err = errors.Wrap(err, "getName")
		return
	}
	defer rows.Close()

	// loop through rows
	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			err = errors.Wrap(err, "getName")
			return
		}
		break
	}

	err = rows.Err()
	if err != nil {
		err = errors.Wrap(err, "getName")
	}
	return
}

func (d *database) getFriendsName(publicKey string) (name string) {
	query := "SELECT sender FROM letters WHERE opened == 1 AND letter_purpose == 'share-key' AND letter_content LIKE '%%" + publicKey + "%%' LIMIT 1;"
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	var sender string
	for rows.Next() {
		rows.Scan(&sender)
	}
	if sender == "" {
		return
	}

	senderName, _ := d.getName(sender)
	if senderName != "" {
		sender = senderName
	}

	return "Friends of " + sender
}

// deleteLetterFromID will delete a letter with the pertaining ID.
func (d *database) deleteLetterFromID(id string) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "deleteLetterFromID")
	}
	query := fmt.Sprintf("DELETE FROM letters WHERE id == '%s'", id)
	log.Debug(query)
	stmt, err := tx.Prepare(query)
	if err != nil {
		return errors.Wrap(err, "deleteLetterFromID")
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		return errors.Wrap(err, "deleteLetterFromID")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "deleteLetterFromID")
	}

	return
}

func (d *database) isReplaced(id string) (yes bool) {
	query := "SELECT id FROM letters WHERE opened == 1 AND letter_replaces=='" + id + "' LIMIT 1;"
	log.Debug(query)
	rows, err := d.db.Query(query)
	if err != nil {
		log.Error(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		return true
	}
	return false
}
