package main

import (
	"reflect"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	type testCase struct {
		expected   ValidNomination
		nomination *nominationInfo
		nominator  *CMSInfo
		office     *officeInfo
	}
	cases := []testCase{
		testCase{
			expected: ValidNomination{Valid: true, Problems: Problems{}},
			nominator: &CMSInfo{
				Type:           "Student",
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "all",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
		},
		testCase{
			expected: ValidNomination{Valid: false, Problems: Problems{"Not a student."}},
			nominator: &CMSInfo{
				Type:           "Staff",
				GraduationDate: cmsDate{Time: time.Time{}},
			},
			office: &officeInfo{Type: "all"},
		},
		testCase{
			expected: ValidNomination{Valid: true, Problems: Problems{}},
			nominator: &CMSInfo{
				Type:           "Student",
				ClassByCredit:  "Junior",
				GraduationDate: createCMSDate("2019-01-01"),
			},
			office: &officeInfo{
				Type: "junior",
				Cohorts: []string{
					"2019",
				},
			},
		},
		testCase{
			expected: ValidNomination{Valid: false, Problems: Problems{"Not Greek-affiliated."}},
			nominator: &CMSInfo{
				Type:           "Student",
				Greek:          false,
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "greek",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
		},
		testCase{
			expected: ValidNomination{Valid: true, Problems: Problems{}},
			nominator: &CMSInfo{
				Type:           "Student",
				Greek:          true,
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "greek",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
		},
		testCase{
			expected: ValidNomination{Valid: false, Problems: Problems{"Greek-affiliated."}},
			nominator: &CMSInfo{
				Type:           "Student",
				Greek:          true,
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "independent",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
		},
		testCase{
			expected: ValidNomination{Valid: true, Problems: Problems{}},
			nominator: &CMSInfo{
				Type:           "Student",
				Greek:          false,
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "independent",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
		},
	}

	for _, c := range cases {
		actual := validate(c.nomination, c.nominator, c.office)

		if actual.Valid != c.expected.Valid || !actual.Problems.equal(c.expected.Problems) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}

func TestStudentValidator(t *testing.T) {
	type testCase struct {
		expected   Problems
		nomination *nominationInfo
		nominator  *CMSInfo
		office     *officeInfo
	}
	cases := []testCase{
		testCase{
			expected: Problems{"Not a student."},
			nominator: &CMSInfo{
				Type: "Employee",
			},
			nomination: nil,
			office:     nil,
		},
	}

	for _, c := range cases {
		actual := studentValidator(c.nomination, c.nominator, c.office)
		if !actual.equal(c.expected) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}

func createCMSDate(s string) cmsDate {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	d := cmsDate{Time: t}
	return d
}

func TestCohortValidator(t *testing.T) {
	type testCase struct {
		expected   Problems
		nomination *nominationInfo
		nominator  *CMSInfo
		office     *officeInfo
	}
	cases := []testCase{
		testCase{
			expected: Problems{"Not an undergraduate student."},
			nominator: &CMSInfo{
				ClassByCredit:  "Graduate",
				GraduationDate: createCMSDate("2018-01-01"),
			},
			office: &officeInfo{
				Type: "undergraduate",
				Cohorts: []string{
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
			nomination: nil,
		},
		testCase{
			expected: Problems{"Not a graduate student."},
			nominator: &CMSInfo{
				ClassByCredit:  "Freshman",
				GraduationDate: createCMSDate("2021-01-01"),
			},
			office: &officeInfo{
				Type: "graduate",
				Cohorts: []string{
					"graduate",
				},
			},
			nomination: nil,
		},
		testCase{
			expected: Problems{"Cohorts not eligible for this office."},
			nominator: &CMSInfo{
				ClassByCredit:  "Junior",
				GraduationDate: createCMSDate("2019-01-01"),
			},
			office: &officeInfo{
				Type: "2020",
				Cohorts: []string{
					"2020",
				},
			},
			nomination: nil,
		},
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				ClassByCredit:  "Junior",
				GraduationDate: createCMSDate("2020-01-01"),
			},
			office: &officeInfo{
				Type: "2020",
				Cohorts: []string{
					"2020",
				},
			},
			nomination: nil,
		},
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				ClassByCredit:  "Fifth-Year",
				GraduationDate: createCMSDate("2018-01-01"),
			},
			office: &officeInfo{
				Type: "2018",
				Cohorts: []string{
					"2018",
				},
			},
			nomination: nil,
		},
	}

	for _, c := range cases {
		actual := cohortValidator(c.nomination, c.nominator, c.office)
		if !actual.equal(c.expected) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}

func TestInitialsValidator(t *testing.T) {
	type testCase struct {
		expected   Problems
		nomination *nominationInfo
		nominator  *CMSInfo
		office     *officeInfo
	}
	cases := []testCase{
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				FirstName: "Sidney",
				LastName:  "Kochman",
			},
			nomination: &nominationInfo{
				Name:     "Sidney Kochman",
				Initials: "SK",
			},
			office: nil,
		},
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				FirstName:  "Sidney",
				MiddleName: "David",
				LastName:   "Kochman",
			},
			nomination: &nominationInfo{
				Name:     "Sidney Kochman",
				Initials: "SDK",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Initials do not match Institute records."},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
			},
			nomination: &nominationInfo{
				Initials: "SK",
			},
			office: nil,
		},
		testCase{
			expected: Problems{
				"Initials do not match Institute records.",
			},
			nominator: &CMSInfo{
				FirstName: "Sidney",
				LastName:  "Kochman",
			},
			nomination: &nominationInfo{
				Name:     "Joseph Lyon",
				Initials: "EZ",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Initials shorter than two characters."},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
			},
			nomination: &nominationInfo{
				Initials: "S",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Initials longer than three characters."},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
			},
			nomination: &nominationInfo{
				Initials: "SDLK",
			},
			office: nil,
		},
	}

	for _, c := range cases {
		actual := initialsValidator(c.nomination, c.nominator, c.office)
		if !actual.equal(c.expected) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}

func TestOfficeInfoFromType(t *testing.T) {
	type testCase struct {
		expected   officeInfo
		officeType string
	}
	cases := []testCase{
		testCase{
			expected: officeInfo{
				Type: "all",
				Cohorts: []string{
					"graduate",
					"2018",
					"2019",
					"2020",
					"2021",
				},
			},
			officeType: "all",
		},
	}

	for _, c := range cases {
		actual := officeInfoFromType(c.officeType)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}

func TestNameValidator(t *testing.T) {
	type testCase struct {
		expected   Problems
		nomination *nominationInfo
		nominator  *CMSInfo
		office     *officeInfo
	}
	cases := []testCase{
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				FirstName:  "Sidney",
				MiddleName: "David",
				LastName:   "Kochman",
			},
			nomination: &nominationInfo{
				Name: "Sidney Kochman",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"First name does not match Institute records."},
			nominator: &CMSInfo{
				FirstName:  "Sidney",
				MiddleName: "David",
				LastName:   "Kochman",
			},
			nomination: &nominationInfo{
				Name: "Joey Kochman",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Last name does not match Institute records."},
			nominator: &CMSInfo{
				FirstName:  "Sidney",
				MiddleName: "David",
				LastName:   "Kochman",
			},
			nomination: &nominationInfo{
				Name: "Sidney Lyon",
			},
			office: nil,
		},
	}

	for _, c := range cases {
		actual := nameValidator(c.nomination, c.nominator, c.office)
		if !actual.equal(c.expected) {
			t.Errorf("expected %+v, got %+v", c.expected, actual)
		}
	}
}
