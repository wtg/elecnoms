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
		testCase{
			expected: ValidNomination{Valid: false, Problems: Problems{"Not a graduate student."}},
			nominator: &CMSInfo{
				Type:           "Student",
				Greek:          true,
				GraduationDate: createCMSDate("2018-01-01"),
			},
			office: &officeInfo{
				Type: "graduate",
				Cohorts: []string{
					"graduate",
				},
			},
		},
	}

	for _, c := range cases {
		actual := validate(c.nomination, c.nominator, c.office, Problems{})

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
				Type: "2021",
				Cohorts: []string{
					"2021",
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

func TestRinRCSMatchValidator(t *testing.T) {
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
				FirstName: "Joseph",
				LastName:  "Lyon",
				RIN:       "661520777",
			},
			nomination: &nominationInfo{
				Name:       "Joseph Lyon",
				RcsID:      "lyonj4",
				PartialRIN: "777",
			},
			office: nil,
		},
		testCase{
			expected: Problems{},
			nominator: &CMSInfo{
				FirstName:  "Sidney",
				MiddleName: "David",
				LastName:   "Kochman",
				RIN:        "6615200999",
			},
			nomination: &nominationInfo{
				Name:       "Sidney Kochman",
				RcsID:      "kochms",
				PartialRIN: "999",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Mismatched RIN digits."},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
				RIN:       "661300999",
			},
			nomination: &nominationInfo{
				Name:       "Joseph Lyon",
				PartialRIN: "998",
			},
			office: nil,
		},
		testCase{
			expected: Problems{
				"Partial RIN value contains more than three digits.", "Mismatched RIN digits.",
			},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
				RIN:       "661402089",
			},
			nomination: &nominationInfo{
				Name:       "Joseph Lyon",
				PartialRIN: "6949",
			},
			office: nil,
		},
		testCase{
			expected: Problems{"Partial RIN value contains more than three digits.", "Mismatched RIN digits."},
			nominator: &CMSInfo{
				FirstName: "Joseph",
				LastName:  "Lyon",
				RIN:       "661402089",
			},
			nomination: &nominationInfo{
				Name:       "Joseph Lyon",
				PartialRIN: "2089",
			},
			office: nil,
		},
	}

	for _, c := range cases {
		actual := rinRCSMatchValidator(c.nomination, c.nominator, c.office)
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
					"2019",
					"2020",
					"2021",
					"2022",
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
