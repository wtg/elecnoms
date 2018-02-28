package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type nominationCount struct {
	OfficeID    int    `json:"office_id"`
	RCSID       string `json:"rcs_id"`
	Nominations int    `json:"nominations"`
}

func nominationCounts(w http.ResponseWriter, r *http.Request) {
	db, err := getDB()
	if err != nil {
		log.Printf("unable to get database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var rows *sql.Rows

	// extract/validate candidate RCS ID
	rcs := r.FormValue("rcs")
	if rcs != "" {
		// only return counts for this RCS ID
		rows, err = db.Query("SELECT rcs_id, office_id, COUNT(*) as nominations FROM nominations WHERE rcs_id = ? AND valid = true GROUP BY rcs_id, office_id;", rcs)
	} else {
		rows, err = db.Query("SELECT rcs_id, office_id, COUNT(*) as nominations FROM nominations WHERE valid = true GROUP BY rcs_id, office_id;")
	}
	if err != nil {
		log.Printf("unable to query database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	nominations := []nominationCount{}
	for rows.Next() {
		nomCount := nominationCount{}
		err := rows.Scan(&nomCount.RCSID, &nomCount.OfficeID, &nomCount.Nominations)
		if err != nil {
			log.Printf("unable to scan: %s", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		nominations = append(nominations, nomCount)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(nominations)
}
