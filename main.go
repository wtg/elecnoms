package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi"
	_ "github.com/go-sql-driver/mysql"
)

type Nomination struct {
	ID     int    `json:"id"`
	RIN    string `json:"rin"`
	RcsID  string `json:"rcs"`
	Valid  *bool  `json:"valid"`
	Page   int    `json:"page"`
	Number int    `json:"number"`
}

type NominationPage struct {
	Number      int          `json:"page_number"`
	Nominations []Nomination `json:"nominations"`
	OfficeID    int          `json:"office_id"`
	Submitted   time.Time    `json:"submitted"`
}

var activeElectionQuery = "(SELECT value FROM configurations WHERE `key` = 'active_election_id')"

// getDB returns a database connection. The caller is responsible for closing it.
func getDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", os.Getenv("DATABASE_URL")+"?parseTime=true")
	if err != nil {
		log.Printf("unable to open database: %s", err.Error())
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		log.Printf("unable to ping database: %s", err.Error())
		return nil, err
	}
	return db, err
}

func getCandidateAssistants(rcs string) ([]string, error) {
	assistants := []string{}
	db, err := getDB()
	if err != nil {
		return assistants, err
	}

	rows, err := db.Query("SELECT rcs_id FROM assistants WHERE candidate_rcs_id = ? AND election_id = "+activeElectionQuery, rcs)
	if err != nil {
		return assistants, err
	}

	for rows.Next() {
		var assistant string
		err = rows.Scan(&assistant)
		if err != nil {
			return assistants, err
		}
		assistants = append(assistants, strings.ToLower(assistant))
	}

	return assistants, nil
}

func contains(slice []string, str string) bool {
	for _, elem := range slice {
		if elem == str {
			return true
		}
	}
	return false
}

// listNominations returns a list of nomination pages for a given RCS ID.
// If an office ID is provided, it only lists nominations for that office.
// Authorization is required, and people with permission are admins, the candidate with the specified RCS ID, and her assistants.
func listNominations(w http.ResponseWriter, r *http.Request) {
	// extract/validate RCS ID
	rcs := strings.ToLower(r.FormValue("rcs"))
	if rcs == "" {
		log.Print("missing rcs")
		http.Error(w, "missing rcs", http.StatusUnprocessableEntity)
		return
	}

	// check if this user has permission to do this
	admin := adminFromContext(r.Context())
	casUser := casUserFromContext(r.Context())
	// find assistants and see if this user is one
	assistants, err := getCandidateAssistants(rcs)
	if err != nil {
		log.Printf("unable to get candidate assistants: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	assistant := contains(assistants, casUser)
	if !admin && casUser != rcs && !assistant {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// query database for nominations
	db, err := getDB()
	if err != nil {
		log.Printf("unable to get database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	var rows *sql.Rows

	// Extract office ID and page number from query string.
	// Page number only makes sense if office is set.
	office := r.FormValue("office")
	pageNumber := r.FormValue("page")
	if office != "" {
		// only return nominations for this office
		if pageNumber != "" {
			// and only return a specific page
			rows, err = db.Query("SELECT nomination_id, nomination_partial_rin, nomination_rcs_id, valid, page, office_id, date, number FROM nominations WHERE rcs_id = ? AND office_id = ? AND page = ? AND election_id = "+activeElectionQuery+" ORDER BY number", rcs, office, pageNumber)
		} else {
			rows, err = db.Query("SELECT nomination_id, nomination_partial_rin, nomination_rcs_id, valid, page, office_id, date, number FROM nominations WHERE rcs_id = ? AND office_id = ? AND election_id = "+activeElectionQuery+" ORDER BY number", rcs, office)
		}
	} else {
		rows, err = db.Query("SELECT nomination_id, nomination_partial_rin, nomination_rcs_id, valid, page, office_id, date, number FROM nominations WHERE rcs_id = ? AND election_id = "+activeElectionQuery+" ORDER BY number", rcs)
	}
	if err != nil {
		log.Printf("unable to query database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// sort nominations into pages (map of office ID to page number to NominationPage)
	offices := map[int]map[int]NominationPage{}
	for rows.Next() {
		nomination := Nomination{}
		var nomOfficeID int
		var date time.Time
		err = rows.Scan(&nomination.ID, &nomination.RIN, &nomination.RcsID, &nomination.Valid, &nomination.Page, &nomOfficeID, &date, &nomination.Number)
		if err != nil {
			log.Printf("unable to scan: %s", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// add nomination to page
		if _, ok := offices[nomOfficeID]; !ok {
			offices[nomOfficeID] = map[int]NominationPage{}
		}
		page := offices[nomOfficeID][nomination.Page]
		page.Number = nomination.Page
		page.Nominations = append(page.Nominations, nomination)
		page.OfficeID = nomOfficeID
		page.Submitted = date
		offices[nomOfficeID][nomination.Page] = page
	}

	// flatten pages from map to list
	flat := []NominationPage{}
	for _, pages := range offices {
		for _, page := range pages {
			flat = append(flat, page)
		}
	}

	// return them in ascending page number order
	sort.Slice(flat, func(i, j int) bool {
		return flat[i].Number < flat[j].Number
	})

	// return as JSON
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(flat)
}

func addNominations(w http.ResponseWriter, r *http.Request) {
	// extract/validate candidate RCS ID
	rcs := strings.ToLower(r.FormValue("rcs"))
	if rcs == "" {
		log.Print("missing RCS")
		http.Error(w, "missing RCS", http.StatusUnprocessableEntity)
		return
	}
	// check if this user has permission to do this
	admin := adminFromContext(r.Context())
	casUser := casUserFromContext(r.Context())
	// find assistants and see if this user is one
	assistants, err := getCandidateAssistants(rcs)
	if err != nil {
		log.Printf("unable to get candidate assistants: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	assistant := contains(assistants, casUser)
	if !admin && casUser != rcs && !assistant {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// extract/validate office ID
	office := r.FormValue("office")
	if office == "" {
		log.Print("missing office")
		http.Error(w, "missing office", http.StatusUnprocessableEntity)
		return
	}

	// decode nominations
	nominations := []Nomination{}
	dec := json.NewDecoder(r.Body)
	err = dec.Decode(&nominations)
	if err != nil {
		log.Printf("unable to decode JSON: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	fmt.Printf("%+v", nominations)

	// sanity check
	if len(nominations) > 25 {
		http.Error(w, "too many nominations; only 25 per page", http.StatusUnprocessableEntity)
		return
	}

	// start database transaction
	db, err := getDB()
	if err != nil {
		log.Printf("unable to get database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		log.Printf("unable to begin transaction: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// figure out the highest existing page number and add 1 to it
	row := tx.QueryRow("SELECT COALESCE(MAX(page), 0) FROM nominations WHERE rcs_id = ? and office_id = ? and election_id = "+activeElectionQuery, rcs, office)
	var prevPage int
	err = row.Scan(&prevPage)
	if err != nil {
		log.Printf("unable to query database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	pageNum := prevPage + 1

	// loop over provided nominations and insert
	for _, nomination := range nominations {

		_, err = tx.Exec("INSERT INTO nominations (rcs_id, office_id, nomination_partial_rin, nomination_rcs_id, page, number, election_id) VALUES (?, ?, ?, ?, ?, ?, "+activeElectionQuery+");", rcs, office, nomination.RIN, strings.ToLower(nomination.RcsID), pageNum, nomination.Number)
		if err != nil {
			log.Printf("unable to query database: %s", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	tx.Commit()
}

// modifyNomination updates an existing nomination to match the provided nomination.
// Primary use of this is for marking nominations valid, invalid, or pending,
// but it can be used to modify almost any information about a nomination.
// Requires authorization, and only admins can use it.
func modifyNomination(w http.ResponseWriter, r *http.Request) {
	// check if this user has permission to do this
	admin := adminFromContext(r.Context())
	if !admin {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// extract/validate nomination ID
	nomID := r.FormValue("nomination")
	if nomID == "" {
		log.Print("missing nomination ID")
		http.Error(w, "missing nomination ID", http.StatusUnprocessableEntity)
		return
	}

	// decode nomination
	nomination := Nomination{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&nomination)
	if err != nil {
		log.Printf("unable to decode JSON: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// get database
	db, err := getDB()
	if err != nil {
		log.Printf("unable to get database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// update nomination in database
	_, err = db.Exec("UPDATE nominations SET nomination_partial_rin = ?, nomination_rcs_id = ?, page = ?, valid = ?, number = ? WHERE nomination_id = ?;", nomination.RIN, nomination.RcsID, nomination.Page, nomination.Valid, nomination.Number, nomination.ID)
	if err != nil {
		log.Printf("unable to query database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(authenticate)
	r.Get("/", listNominations)
	r.Post("/", addNominations)
	r.Put("/", modifyNomination)
	r.Get("/validate", validateNomination)
	r.Get("/counts", nominationCounts)

	listenURL := os.Getenv("LISTEN_URL")
	if listenURL == "" {
		listenURL = "0.0.0.0:3001"
	}
	log.Printf("elecnoms listening on %s...", listenURL)
	log.Fatal(http.ListenAndServe(listenURL, r))
}
