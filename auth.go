package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// sessionData represents session data stored in the database
type sessionData struct {
	CASUser       string `json:"cas_user"`
	Admin         bool   `json:"admin_rights"`
	Authenticated bool   `json:"is_authenticated"`
}

// Key for context, set in authetnicate middleware. Go docs say,
// "Users of WithValue should define their own types for keys."
type contextKey string

// context keys
const casUserKey = contextKey("casUser")
const adminKey = contextKey("admin")
const authenticatedKey = contextKey("authenticated")

// Take cookie, extract session ID, decode additional info from database,
// and attach to context. Assumes that cookie has been validated already.
func contextFromCookie(ctx context.Context, cookie *http.Cookie) (context.Context, error) {
	// extract stuff from cookie
	messageSplit := strings.Split(cookie.Value, ".")
	sessionID := []byte(messageSplit[0][4:])

	db, err := getDB()
	if err != nil {
		return ctx, err
	}
	defer db.Close()

	row := db.QueryRow("SELECT data FROM sessions WHERE session_id = ?", sessionID)
	var jsonData []byte
	err = row.Scan(&jsonData)
	if err != nil {
		return ctx, err
	}
	sd := sessionData{}
	err = json.Unmarshal(jsonData, &sd)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, casUserKey, strings.ToLower(sd.CASUser))
	ctx = context.WithValue(ctx, adminKey, sd.Admin)
	ctx = context.WithValue(ctx, authenticatedKey, sd.Authenticated)
	return ctx, nil
}

func unauthenticatedContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, casUserKey, "")
	ctx = context.WithValue(ctx, adminKey, false)
	ctx = context.WithValue(ctx, authenticatedKey, false)
	return ctx
}

// context accessors
func adminFromContext(ctx context.Context) bool {
	admin, ok := ctx.Value(adminKey).(bool)
	if !ok {
		return false
	}
	return admin
}
func casUserFromContext(ctx context.Context) string {
	casUser, ok := ctx.Value(casUserKey).(string)
	if !ok {
		return ""
	}
	return casUser
}

// verifyCookie returns whether or not a cookie signed with https://github.com/tj/node-cookie-signature
// has been modified, tampered with, or otherwise mangled.
func verifyCookie(cookie *http.Cookie) (bool, error) {
	// extract stuff from cookie
	messageSplit := strings.Split(cookie.Value, ".")
	message := []byte(messageSplit[0][4:])
	messageMAC := messageSplit[1]

	// create HMAC to see if it matches the one in the cookie
	mac := hmac.New(sha256.New, []byte(os.Getenv("SESSION_SECRET")))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)
	expectedMACEncoded := base64.RawStdEncoding.EncodeToString(expectedMAC)
	messageMACUnescaped, err := url.QueryUnescape(messageMAC)
	if err != nil {
		return false, err
	}

	return messageMACUnescaped == expectedMACEncoded, nil
}

// authenticate decodes a session cookie from https://github.com/expressjs/session,
// extracts the session info from the database, and stores it on the request context.
func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// This is a function so that we return the most recent version of r,
			// which is assigned to multiple times in this handler.
			next.ServeHTTP(w, r)
		}()

		// save original context, and replace request context with unauthenticated context
		origCtx := r.Context()
		r = r.WithContext(unauthenticatedContext(origCtx))

		// this is what Express sessions names cookies by default
		cookie, err := r.Cookie("connect.sid")
		if err == http.ErrNoCookie {
			return
		}

		valid, err := verifyCookie(cookie)
		if err != nil {
			log.Printf("unable to verify cookie: %s", err.Error())
			return
		}
		if !valid {
			log.Printf("cookie invalid")
			return
		}

		// extract cookie info and attach to request context, ignoring unauthenticated context
		ctx, err := contextFromCookie(origCtx, cookie)
		if err != nil {
			log.Printf("unable to attach session info to context: %s", err.Error())
			return
		}
		r = r.WithContext(ctx)
	})
}
