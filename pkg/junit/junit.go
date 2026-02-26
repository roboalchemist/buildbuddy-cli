package junit

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// TestSuites represents the root element of a JUnit XML file.
type TestSuites struct {
	XMLName    xml.Name    `xml:"testsuites" json:"-"`
	Name       string      `xml:"name,attr" json:"name,omitempty"`
	Tests      int         `xml:"tests,attr" json:"tests"`
	Failures   int         `xml:"failures,attr" json:"failures"`
	Errors     int         `xml:"errors,attr" json:"errors"`
	Time       float64     `xml:"time,attr" json:"time"`
	TestSuites []TestSuite `xml:"testsuite" json:"testSuites,omitempty"`
}

// TestSuite represents a test suite.
type TestSuite struct {
	XMLName   xml.Name   `xml:"testsuite" json:"-"`
	Name      string     `xml:"name,attr" json:"name,omitempty"`
	Tests     int        `xml:"tests,attr" json:"tests"`
	Failures  int        `xml:"failures,attr" json:"failures"`
	Errors    int        `xml:"errors,attr" json:"errors"`
	Skipped   int        `xml:"skipped,attr" json:"skipped"`
	Time      float64    `xml:"time,attr" json:"time"`
	Timestamp string     `xml:"timestamp,attr" json:"timestamp,omitempty"`
	TestCases []TestCase `xml:"testcase" json:"testCases,omitempty"`
}

// TestCase represents a single test case.
type TestCase struct {
	XMLName   xml.Name `xml:"testcase" json:"-"`
	Name      string   `xml:"name,attr" json:"name"`
	Classname string   `xml:"classname,attr" json:"classname,omitempty"`
	Time      float64  `xml:"time,attr" json:"time"`
	Status    string   `json:"status"`
	Failure   *Failure `xml:"failure,omitempty" json:"failure,omitempty"`
	Error     *Error   `xml:"error,omitempty" json:"error,omitempty"`
	Skipped   *Skipped `xml:"skipped,omitempty" json:"skipped,omitempty"`
}

// Failure represents a test failure.
type Failure struct {
	Message string `xml:"message,attr" json:"message,omitempty"`
	Type    string `xml:"type,attr" json:"type,omitempty"`
	Content string `xml:",chardata" json:"content,omitempty"`
}

// Error represents a test error.
type Error struct {
	Message string `xml:"message,attr" json:"message,omitempty"`
	Type    string `xml:"type,attr" json:"type,omitempty"`
	Content string `xml:",chardata" json:"content,omitempty"`
}

// Skipped represents a skipped test.
type Skipped struct {
	Message string `xml:"message,attr" json:"message,omitempty"`
}

// Parse parses a JUnit XML string and returns a TestSuites object.
// It tries parsing as <testsuites> first, falls back to single <testsuite>.
func Parse(xmlContent string) (*TestSuites, error) {
	var testsuites TestSuites

	err := xml.Unmarshal([]byte(xmlContent), &testsuites)
	if err == nil && testsuites.XMLName.Local == "testsuites" {
		computeStats(&testsuites)
		setStatuses(&testsuites)
		return &testsuites, nil
	}

	var testsuite TestSuite
	err = xml.Unmarshal([]byte(xmlContent), &testsuite)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JUnit XML: %w", err)
	}

	testsuites = TestSuites{
		Name:       testsuite.Name,
		Tests:      testsuite.Tests,
		Failures:   testsuite.Failures,
		Errors:     testsuite.Errors,
		Time:       testsuite.Time,
		TestSuites: []TestSuite{testsuite},
	}

	computeStats(&testsuites)
	setStatuses(&testsuites)
	return &testsuites, nil
}

// computeStats computes or validates statistics from the actual test cases.
// Fills in missing summary attributes from actual test case data.
func computeStats(testsuites *TestSuites) {
	totalTests := 0
	totalFailures := 0
	totalErrors := 0
	totalTime := 0.0

	for i := range testsuites.TestSuites {
		suite := &testsuites.TestSuites[i]

		suiteTests := 0
		suiteFailures := 0
		suiteErrors := 0
		suiteSkipped := 0
		suiteTime := 0.0

		for _, tc := range suite.TestCases {
			suiteTests++
			suiteTime += tc.Time
			if tc.Failure != nil {
				suiteFailures++
			}
			if tc.Error != nil {
				suiteErrors++
			}
			if tc.Skipped != nil {
				suiteSkipped++
			}
		}

		if suite.Tests == 0 {
			suite.Tests = suiteTests
		}
		if suite.Failures == 0 && suiteFailures > 0 {
			suite.Failures = suiteFailures
		}
		if suite.Errors == 0 && suiteErrors > 0 {
			suite.Errors = suiteErrors
		}
		if suite.Skipped == 0 && suiteSkipped > 0 {
			suite.Skipped = suiteSkipped
		}
		if suite.Time == 0 {
			suite.Time = suiteTime
		}

		totalTests += suite.Tests
		totalFailures += suite.Failures
		totalErrors += suite.Errors
		totalTime += suite.Time
	}

	if testsuites.Tests == 0 {
		testsuites.Tests = totalTests
	}
	if testsuites.Failures == 0 && totalFailures > 0 {
		testsuites.Failures = totalFailures
	}
	if testsuites.Errors == 0 && totalErrors > 0 {
		testsuites.Errors = totalErrors
	}
	if testsuites.Time == 0 {
		testsuites.Time = totalTime
	}
}

// setStatuses populates the Status field on each TestCase for JSON output.
func setStatuses(testsuites *TestSuites) {
	for i := range testsuites.TestSuites {
		for j := range testsuites.TestSuites[i].TestCases {
			tc := &testsuites.TestSuites[i].TestCases[j]
			switch {
			case tc.Failure != nil:
				tc.Status = "FAILED"
			case tc.Error != nil:
				tc.Status = "ERROR"
			case tc.Skipped != nil:
				tc.Status = "SKIPPED"
			default:
				tc.Status = "PASSED"
			}
		}
	}
}

// Summary returns a compact summary struct suitable for JSON output.
type Summary struct {
	Name     string         `json:"name,omitempty"`
	Tests    int            `json:"tests"`
	Failures int            `json:"failures"`
	Errors   int            `json:"errors"`
	Time     float64        `json:"time"`
	Passed   int            `json:"passed"`
	Skipped  int            `json:"skipped"`
	Suites   []SuiteSummary `json:"suites,omitempty"`
}

// SuiteSummary is a per-suite summary for JSON output.
type SuiteSummary struct {
	Name      string     `json:"name,omitempty"`
	Tests     int        `json:"tests"`
	Failures  int        `json:"failures"`
	Errors    int        `json:"errors"`
	Skipped   int        `json:"skipped"`
	Time      float64    `json:"time"`
	TestCases []TestCase `json:"testCases,omitempty"`
}

// ToSummary returns a Summary struct for JSON rendering.
func (ts *TestSuites) ToSummary() Summary {
	s := Summary{
		Name:     ts.Name,
		Tests:    ts.Tests,
		Failures: ts.Failures,
		Errors:   ts.Errors,
		Time:     ts.Time,
	}

	totalSkipped := 0
	for _, suite := range ts.TestSuites {
		totalSkipped += suite.Skipped
		s.Suites = append(s.Suites, SuiteSummary{
			Name:      suite.Name,
			Tests:     suite.Tests,
			Failures:  suite.Failures,
			Errors:    suite.Errors,
			Skipped:   suite.Skipped,
			Time:      suite.Time,
			TestCases: suite.TestCases,
		})
	}
	s.Skipped = totalSkipped
	s.Passed = s.Tests - s.Failures - s.Errors - s.Skipped
	if s.Passed < 0 {
		s.Passed = 0
	}
	return s
}

// FormatOutput formats the test results in a human-readable way.
func FormatOutput(testsuites *TestSuites) string {
	var out strings.Builder

	printf := func(format string, args ...interface{}) {
		out.WriteString(fmt.Sprintf(format, args...))
	}

	printf("\nTest Summary\n")
	printf("-----------------------------------------------------\n")
	printf("Total Tests:  %d\n", testsuites.Tests)
	printf("Failures:     %d\n", testsuites.Failures)
	printf("Errors:       %d\n", testsuites.Errors)
	printf("Duration:     %.2fs\n", testsuites.Time)
	printf("-----------------------------------------------------\n")

	var failed, errored, passed []TestCase
	for _, suite := range testsuites.TestSuites {
		for _, tc := range suite.TestCases {
			if tc.Failure != nil {
				failed = append(failed, tc)
			} else if tc.Error != nil {
				errored = append(errored, tc)
			} else if tc.Skipped == nil {
				passed = append(passed, tc)
			}
		}
	}

	formatFailedTests(&out, failed)
	formatErroredTests(&out, errored)
	formatPassedTests(&out, passed)

	printf("\n")
	return out.String()
}

func formatFailedTests(out *strings.Builder, failed []TestCase) {
	if len(failed) == 0 {
		return
	}
	printf := func(format string, args ...interface{}) {
		out.WriteString(fmt.Sprintf(format, args...))
	}

	printf("\nFailed Tests (%d):\n", len(failed))
	for i, tc := range failed {
		printf("\n%d. %s\n", i+1, tc.Name)
		if tc.Classname != "" {
			printf("   Class: %s\n", tc.Classname)
		}
		printf("   Duration: %.2fs\n", tc.Time)
		if tc.Failure != nil {
			if tc.Failure.Message != "" {
				printf("   Message: %s\n", tc.Failure.Message)
			}
			if tc.Failure.Content != "" {
				lines := strings.Split(strings.TrimSpace(tc.Failure.Content), "\n")
				printf("   Details:\n")
				for _, line := range lines {
					printf("     %s\n", line)
				}
			}
		}
	}
}

func formatErroredTests(out *strings.Builder, errored []TestCase) {
	if len(errored) == 0 {
		return
	}
	printf := func(format string, args ...interface{}) {
		out.WriteString(fmt.Sprintf(format, args...))
	}

	printf("\nErrored Tests (%d):\n", len(errored))
	for i, tc := range errored {
		printf("\n%d. %s\n", i+1, tc.Name)
		if tc.Classname != "" {
			printf("   Class: %s\n", tc.Classname)
		}
		printf("   Duration: %.2fs\n", tc.Time)
		if tc.Error != nil {
			if tc.Error.Message != "" {
				printf("   Message: %s\n", tc.Error.Message)
			}
			if tc.Error.Content != "" {
				lines := strings.Split(strings.TrimSpace(tc.Error.Content), "\n")
				printf("   Details:\n")
				for _, line := range lines {
					printf("     %s\n", line)
				}
			}
		}
	}
}

func formatPassedTests(out *strings.Builder, passed []TestCase) {
	if len(passed) == 0 {
		return
	}
	printf := func(format string, args ...interface{}) {
		out.WriteString(fmt.Sprintf(format, args...))
	}

	printf("\nPassed Tests (%d)\n", len(passed))
	limit := 5
	if len(passed) < limit {
		limit = len(passed)
	}
	for i := 0; i < limit; i++ {
		printf("   - %s (%.2fs)\n", passed[i].Name, passed[i].Time)
	}
	if len(passed) > limit {
		printf("   ... and %d more\n", len(passed)-limit)
	}
}
