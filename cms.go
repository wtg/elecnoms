package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var errInfoNotFound = errors.New("RIN not found")

type cmsDate struct {
	time.Time
}

type CMSInfo struct {
	Type           string  `json:"user_type"`
	GraduationDate cmsDate `json:"grad_date"`
	EntryDate      cmsDate `json:"entry_date"`
	ClassByCredit  string  `json:"class_by_credit"`
	Greek          bool    `json:"greek_affiliated"`
	FirstName      string  `json:"first_name"`
	MiddleName     string  `json:"middle_name"`
	LastName       string  `json:"last_name"`
	RIN            string  `json:"student_id"`
}

// we need to include output of class and credit cohort methods
func (c *CMSInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type         string `json:"type"`
		Greek        bool   `json:"greek"`
		FirstName    string `json:"first_name"`
		MiddleName   string `json:"middle_name"`
		LastName     string `json:"last_name"`
		CreditCohort string `json:"credit_cohort"`
		EntryCohort  string `json:"entry_cohort"`
		IsGraduate   bool `json:"is_graduate"`
	}{
		Type:         c.Type,
		Greek:        c.Greek,
		FirstName:    c.FirstName,
		MiddleName:   c.MiddleName,
		LastName:     c.LastName,
		CreditCohort: c.creditCohort(),
		EntryCohort:  c.entryCohort(),
		IsGraduate:   c.graduate(),
	})
}

var cohortOffsets = map[string]int{
	"senior":    0,
	"junior":    1,
	"sophomore": 2,
	"freshman":  3,
}

func (d *cmsDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	t, err := time.Parse("\"2006-01-02\"", s)
	d.Time = t
	return err
}

func (c *CMSInfo) undergraduate() bool {
	if _, ok := cohortOffsets[strings.ToLower(c.ClassByCredit)]; ok {
		return true
	}
	return false
}

func (c *CMSInfo) graduate() bool {
	return strings.ToLower(c.ClassByCredit) == "graduate"
}

func (c *CMSInfo) creditCohort() string {
	year := time.Now().Year()
	cohortYear := year + cohortOffsets[strings.ToLower(c.ClassByCredit)]

	return strconv.Itoa(cohortYear)
}

func (c *CMSInfo) entryCohort() string {
	cohortYear := c.GraduationDate.Format("2006")
	return cohortYear
}

func getCMSInfo(url string) (CMSInfo, error) {
	info := CMSInfo{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Authorization", "Token "+os.Getenv("CMS_TOKEN"))
	c := http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return info, err
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return info, err
		}
		e := errors.New(fmt.Sprintf("unexpected status code: %d, body: %s", resp.StatusCode, body))
		return info, e
	}

	// CMS always returns a response without Content-Length. The only way to determine
	// if the RIN/RCS ID is valid is try to read the body. If empty, it is invalid.
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&info)
	if err == io.EOF {
		return info, errInfoNotFound
	}
	return info, err
}

func cmsInfoRCS(rcs string) (CMSInfo, error) {
	url := fmt.Sprintf("https://cms.union.rpi.edu/api/users/view_rcs/%s/", rcs)
	return getCMSInfo(url)
}

func cmsInfoRIN(rin int) (CMSInfo, error) {
	url := fmt.Sprintf("https://cms.union.rpi.edu/api/users/view_rin/%d/", rin)
	return getCMSInfo(url)
}
