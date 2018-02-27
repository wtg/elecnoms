package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type validationResponse struct {
	Validation *ValidNomination `json:"validation"`
	Office     *officeInfo      `json:"office"`
	Nominator  *CMSInfo         `json:"nominator"`
}

type ValidNomination struct {
	Valid    bool     `json:"valid"`
	Problems Problems `json:"problems,omitempty"`
}

type officeInfo struct {
	ID      int      `json:"id"`
	Type    string   `json:"type"`
	Cohorts []string `json:"cohorts"`
}

type nominationInfo struct {
	RIN          int
	Name         string
	Initials     string
	ID           int
	CandidateRCS string
}

type Validator func(*nominationInfo, *CMSInfo, *officeInfo) Problems
type Problem string
type Problems []Problem

// equal returns whether all elements are shared (order doesn't matter)
func (p Problems) equal(other Problems) bool {
	if len(p) != len(other) {
		return false
	}

	for _, problem := range p {
		found := false
		for i, otherProblem := range other {
			if problem == otherProblem {
				found = true
				other = append(other[:i], other[i+1:]...)
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func studentValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) Problems {
	problems := Problems{}

	// check if nominator is student
	if nominator.Type != "Student" {
		problems = append(problems, "Not a student.")
	}

	return problems
}

func cohortValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) Problems {
	problems := Problems{}

	if nominator == nil || office == nil || nominator.GraduationDate.IsZero() {
		return problems
	}

	// undergrad and grad students
	if strings.ToLower(office.Type) == "undergraduate" && !nominator.undergraduate() {
		problems = append(problems, "Not an undergraduate student.")
		return problems
	}
	if strings.ToLower(office.Type) == "graduate" && !nominator.graduate() {
		problems = append(problems, "Not a graduate student.")
		return problems
	}

	// class year
	found := false
	for _, cohort := range office.Cohorts {
		if cohort == nominator.entryCohort() || cohort == nominator.creditCohort() || (nominator.Greek && cohort == "greek") || (!nominator.Greek && cohort == "independent") {
			found = true
			break
		}
	}
	if !found {
		problems = append(problems, "Cohorts not eligible for this office.")
	}

	return problems
}

func greekIndependentValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) Problems {
	problems := Problems{}

	// Greek
	if strings.ToLower(office.Type) == "greek" && !nominator.Greek {
		problems = append(problems, "Not Greek-affiliated.")
	}

	// Independent
	if strings.ToLower(office.Type) == "independent" && nominator.Greek {
		problems = append(problems, "Greek-affiliated.")
	}

	return problems
}

func initialsValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) Problems {
	problems := Problems{}

	if nomination == nil || nominator == nil {
		return problems
	}

	// length
	if len(nomination.Initials) < 2 {
		problems = append(problems, "Initials shorter than two characters.")
	}
	if len(nomination.Initials) > 3 {
		problems = append(problems, "Initials longer than three characters.")
	}

	if len(nomination.Initials) == 2 {
		initials := strings.ToLower(nomination.Initials)

		// compare to CMS info
		cmsInitials := strings.ToLower(string(nominator.FirstName[0]) + string(nominator.LastName[0]))
		if initials[0] != cmsInitials[0] || initials[1] != cmsInitials[1] {
			problems = append(problems, "Initials do not match Institute records.")
		}
	} else if len(nomination.Initials) == 3 {
		initials := strings.ToLower(nomination.Initials)

		// compare to CMS info
		cmsInitials := strings.ToLower(string(nominator.FirstName[0]) + string(nominator.MiddleName[0]) + string(nominator.LastName[0]))
		if initials[0] != cmsInitials[0] || initials[1] != cmsInitials[1] || initials[2] != cmsInitials[2] {
			problems = append(problems, "Initials do not match Institute records.")
		}
	}

	return problems
}

// nameValidator assumes format "Firstname Lastname", which is super limited and does not
// properly handle everyone's names. This is not currently in use, as the site does not
// collect names of nominators.
func nameValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) Problems {
	problems := Problems{}

	if nomination == nil || nominator == nil {
		return problems
	}

	if len(nomination.Name) == 0 {
		problems = append(problems, "No name provided.")
		return problems
	}

	splitName := strings.Split(nomination.Name, " ")
	if len(splitName) != 2 {
		problems = append(problems, "Name not in recognized format.")
		return problems
	}

	firstName := strings.ToLower(splitName[0])
	lastName := strings.ToLower(splitName[1])

	if firstName != strings.ToLower(nominator.FirstName) {
		problems = append(problems, "First name does not match Institute records.")
	}
	if lastName != strings.ToLower(nominator.LastName) {
		problems = append(problems, "Last name does not match Institute records.")
	}

	return problems
}

// uniqueValidator checks for any other nominations that have the same RIN. It returns problems if another
// nomination has a lower ID than this one (and therefore it is not the only one).
// Because it needs database access, this validator needs to be called differently from the others,
// and it can return an error.
func uniqueValidator(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo) (Problems, error) {
	problems := Problems{}

	if nomination == nil {
		return problems, nil
	}

	db, err := getDB()
	if err != nil {
		return problems, err
	}
	defer db.Close()

	var count int
	row := db.QueryRow("SELECT count(*) FROM nominations WHERE rcs_id = ? AND office_id = ? AND nomination_rin = ? AND nomination_id < ?", nomination.CandidateRCS, office.ID, nomination.RIN, nomination.ID)
	err = row.Scan(&count)
	if err != nil {
		return problems, err
	}

	if count > 0 {
		problems = append(problems, "Nominator has already nominated this candidate for this office.")
	}

	return problems, nil
}

// validate uses election-specific info validators to validate the provided information.
// It takes in existing Problems (may be empty), and it returns a ValidNomination struct.
func validate(nomination *nominationInfo, nominator *CMSInfo, office *officeInfo, problems Problems) ValidNomination {
	validators := []Validator{
		studentValidator,
		cohortValidator,
		greekIndependentValidator,
		initialsValidator,
	}

	for _, validator := range validators {
		problems = append(problems, validator(nomination, nominator, office)...)
	}

	vn := ValidNomination{}
	if len(problems) == 0 {
		vn.Valid = true
	} else {
		vn.Valid = false
	}
	vn.Problems = problems

	return vn
}

// validateNomination returns information about whether a nomination is valid or invalid.
// It requires authorization, and only admins have permission to use it.
// TODO: check if the nomination is a duplicate of an existing one
func validateNomination(w http.ResponseWriter, r *http.Request) {
	// check if this user has permission to do this
	admin := adminFromContext(r.Context())
	if !admin {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// validate provided input
	office := r.FormValue("office")
	if office == "" {
		http.Error(w, "missing office", http.StatusUnprocessableEntity)
		return
	}
	candidateRCS := r.FormValue("candidate_rcs")
	if candidateRCS == "" {
		http.Error(w, "missing candidate RCS", http.StatusUnprocessableEntity)
		return
	}

	// start filling out nomination info fields
	rin, err := strconv.ParseInt(r.FormValue("rin"), 10, 64)
	if err != nil {
		log.Printf("unable to parse int: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}
	nomID, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		log.Printf("unable to parse int: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}
	nomination := nominationInfo{}
	nomination.RIN = int(rin)
	nomination.ID = int(nomID)
	nomination.Initials = r.FormValue("initials")
	nomination.CandidateRCS = candidateRCS

	// get office info
	db, err := getDB()
	if err != nil {
		log.Printf("unable to get database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	row := db.QueryRow("SELECT type FROM offices WHERE office_id = ? AND election_id = "+activeElectionQuery, office)
	if err != nil {
		log.Printf("unable to query database: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var officeType string
	err = row.Scan(&officeType)
	if err == sql.ErrNoRows {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("unable to scan: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	officeInfo := officeInfoFromType(officeType)
	officeID, err := strconv.ParseInt(office, 10, 64)
	if err != nil {
		log.Printf("unable to parse int: %s", err.Error())
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}
	officeInfo.ID = int(officeID)

	nominator, err := cmsInfoRIN(nomination.RIN)
	if err == errInfoNotFound {
		vn := ValidNomination{Valid: false, Problems: Problems{"Invalid RIN."}}
		resp := validationResponse{
			Validation: &vn,
			Office:     &officeInfo,
			Nominator:  nil,
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		err = enc.Encode(resp)
		if err != nil {
			log.Printf("unable to encode JSON: %s", err.Error())
			return
		}
		return
	}

	if err != nil {
		log.Printf("unable to get CMS info: %s", err.Error())
		http.Error(w, "unable to get CMS info", http.StatusInternalServerError)
		return
	}

	// special handling of uniqueValidator
	uniqueProblems, err := uniqueValidator(&nomination, &nominator, &officeInfo)
	if err != nil {
		log.Printf("unable to get CMS info: %s", err.Error())
		http.Error(w, "unable to get CMS info", http.StatusInternalServerError)
		return
	}
	// validate the nomination
	vn := validate(&nomination, &nominator, &officeInfo, uniqueProblems)
	resp := validationResponse{
		Validation: &vn,
		Office:     &officeInfo,
		Nominator:  &nominator,
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		log.Printf("unable to encode JSON: %s", err.Error())
		return
	}
}

func officeInfoFromType(officeType string) officeInfo {
	o := officeInfo{Type: strings.ToLower(officeType)}

	year := time.Now().Year()

	if o.Type == "all" {
		o.Cohorts = []string{"graduate"}
		for i := 0; i < 4; i++ {
			cohort := strconv.FormatInt(int64(year+i), 10)
			o.Cohorts = append(o.Cohorts, cohort)
		}
	} else if o.Type == "greek" {
		o.Cohorts = []string{"greek"}
	} else if o.Type == "independent" {
		o.Cohorts = []string{"independent"}
	} else if o.Type == "graduate" {
		o.Cohorts = []string{"graduate"}
	} else if o.Type == "undergraduate" {
		for i := 0; i < 4; i++ {
			cohort := strconv.FormatInt(int64(year+i), 10)
			o.Cohorts = append(o.Cohorts, cohort)
		}
	} else {
		o.Cohorts = []string{o.Type}
		return o
	}

	return o
}
