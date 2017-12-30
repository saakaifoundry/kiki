package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/schollz/kiki/src/kiki"
	"github.com/schollz/kiki/src/logging"
	"github.com/schollz/kiki/src/person"
)

func init() {
	logging.Setup()
}

var (
	// Port defines what port the carrier should listen on
	Port = "8003"
)

// Run will start the server listening
func Run() {
	// Startup server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.HEAD("/", func(c *gin.Context) { // handler for the uptime robot
		c.String(http.StatusOK, "OK")
	})
	r.POST("/letter", handlerLetter)
	r.POST("/assign", handlerAssign)
	r.POST("/open", handlerOpen)
	r.Run(":" + Port) // listen and serve on 0.0.0.0:Port
}

func respondWithJSON(c *gin.Context, message string, err error) {
	success := err == nil
	if !success {
		message = err.Error()
	}
	c.JSON(http.StatusOK, gin.H{"success": success, "message": message})
}

func handlerOpen(c *gin.Context) {
	respondWithJSON(c, "opened envelopes", handleOpen(c))
}

func handleOpen(c *gin.Context) (err error) {
	err = kiki.OpenEnvelopes()
	if err != nil {
		return
	}
	err = kiki.RegenerateFeed()
	return
}

func handlerAssign(c *gin.Context) {
	respondWithJSON(c, "assigned", handleAssign(c))
}

func handleAssign(c *gin.Context) (err error) {
	assignmentType := c.PostForm("assign")
	assignData := c.PostForm("data")
	if len(assignData) == 0 {
		return errors.New("assigned data cannot be empty")
	}

	return kiki.PostMessage("assign-"+assignmentType, assignData, true)
}

func handlerLetter(c *gin.Context) {
	respondWithJSON(c, "letter added", handleLetter(c))
}

func handleLetter(c *gin.Context) (err error) {
	// get message
	file, err := c.FormFile("message")
	if err != nil {
		return
	}
	data, err := readFormFile(file)
	if err != nil {
		return
	}
	message := string(data)

	// is public
	isPublic := c.PostForm("public") == "yes"

	// get recipients
	recipients := []*person.Person{}
	recipientsString := c.PostForm("recipients")
	if recipientsString != "" {
		var recipientPublicKeys []string
		err = json.Unmarshal([]byte(recipientsString), &recipientPublicKeys)
		if err != nil {
			return
		}
		for _, recipientPublicKeyString := range recipientPublicKeys {
			otherRecipient, err := person.FromPublicKey(recipientPublicKeyString)
			if err != nil {
				logging.Log.Infof("not a valid public key: '%s'", recipientPublicKeyString)
				continue
			}
			recipients = append(recipients, otherRecipient)
		}
	}
	if len(message) == 0 {
		return errors.New("message cannot be empty")
	}
	return kiki.PostMessage("post", message, isPublic, recipients...)
}

func readFormFile(file *multipart.FileHeader) (data []byte, err error) {
	src, err := file.Open()
	if err != nil {
		return
	}
	defer src.Close()
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, src)
	data = buf.Bytes()
	return
}