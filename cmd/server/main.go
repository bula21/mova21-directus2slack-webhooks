package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// common to all webhook JSON bodies Directus sends
type commonFields struct {
	ID         interface{} `json:"id"`
	ModifiedBy int         `json:"modified_by"`
	ModifiedOn string      `json:"modified_on"`
}

// parsed incoming webhook from Directus
type incoming struct {
	common         commonFields
	changes        map[string]interface{}
	slackUrlPath   string
	objectTypeName string
}

func parseIncoming(r *http.Request) (*incoming, error) {
	bodyRaw := json.RawMessage{}
	if err := json.NewDecoder(r.Body).Decode(&bodyRaw); err != nil {
		return nil, err
	}

	common := commonFields{}
	if err := json.Unmarshal(bodyRaw, &common); err != nil {
		return nil, err
	}

	changes := map[string]interface{}{}
	if err := json.Unmarshal(bodyRaw, &changes); err != nil {
		return nil, err
	}
	delete(changes, "id")
	delete(changes, "modified_by")
	delete(changes, "modified_on")

	objectTypeName := strings.TrimSpace(chi.URLParam(r, "objectTypeName"))
	if objectTypeName == "" {
		return nil, errors.New("empty object type name")
	}

	slackUrlPath := strings.TrimSpace(chi.URLParam(r, "*"))
	if slackUrlPath == "" || strings.Count(slackUrlPath, "/") != 2 {
		return nil, errors.New("empty or invalid Slack URL path")
	}

	return &incoming{
		common:         common,
		changes:        changes,
		slackUrlPath:   slackUrlPath,
		objectTypeName: objectTypeName,
	}, nil
}

// https://api.slack.com/reference/surfaces/formatting
func escapeSlackText(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return strings.ReplaceAll(s, "&", "&amp;")
}

func buildOutgoing(in incoming) ([]byte, error) {
	modifiedBy := fmt.Sprintf("%s/admin/#/_/users/%d", directusBaseURL, in.common.ModifiedBy)
	modifiedObj := fmt.Sprintf("%s/admin/#/_/collections/%s/%s", directusBaseURL, in.objectTypeName, in.common.ID)

	text := fmt.Sprintf("<%s|Der User mit ID %d> hat der/die/das <%s|%s mit ID %s> erstellt/editiert/gelöscht.",
		modifiedBy,
		in.common.ModifiedBy,
		escapeSlackText(modifiedObj),
		escapeSlackText(cases.Title(language.Dutch).String(in.objectTypeName)),
		escapeSlackText(fmt.Sprint(in.common.ID)))

	text += fmt.Sprintf("\n\nLinks zum copy-pasten:\n`%s`\n`%s`\n", modifiedBy, escapeSlackText(modifiedObj))

	text += "\n\n*Änderungen*:\n"
	for key, val := range in.changes {
		text += fmt.Sprintf("- _%s_: %s\n", escapeSlackText(key), fmt.Sprint(val))
	}

	body := map[string]string{
		"text": text,
	}
	return json.Marshal(body)
}

func sendOutgoing(slackURL string, outgoing []byte) error {
	request, err := http.NewRequest("POST", slackURL, bytes.NewBuffer(outgoing))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		return err
	}

	ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode/100 != 2 {
		return errors.New("received non-2XX status code")
	}

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userKey := chi.URLParam(r, "userKey")

	if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(userKey)); err != nil {
		log.Print("invalid key")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	incoming, err := parseIncoming(r)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	outgoing, err := buildOutgoing(*incoming)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// send outgoing async
	go func() {
		slackURL := fmt.Sprintf("https://hooks.slack.com/services/%s", incoming.slackUrlPath)
		err = sendOutgoing(slackURL, outgoing)
		if err != nil {
			log.Print(err)
		}
	}()

	w.WriteHeader(http.StatusNoContent)
}

// global CLI options
var (
	keyHash         string
	directusBaseURL string
)

func main() {
	keyHash = os.Getenv("KEY_HASH")
	if keyHash == "" {
		panic("missing KEY_HASH")
	}
	directusBaseURL = strings.TrimRight(os.Getenv("DIRECTUS_BASE_URL"), "/")
	if directusBaseURL == "" {
		directusBaseURL = "https://log.bula21.ch"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "0.0.0.0"
	}
	listen := fmt.Sprintf("%s:%s", addr, port)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 2))
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.CleanPath)
	r.Use(middleware.NoCache)
	r.Use(middleware.Throttle(50))

	r.Post("/{userKey}/{objectTypeName}/*", handler)

	log.Printf("Listening on %s", listen)

	http.ListenAndServe(listen, r)
}
