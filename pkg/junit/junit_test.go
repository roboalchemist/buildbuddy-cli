package junit

import (
	"strings"
	"testing"
)

func TestParse_SingleTestSuite(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="MyTest" tests="2" failures="1" errors="0" time="1.5">
  <testcase name="test_success" classname="MyClass" time="0.5"/>
  <testcase name="test_failure" classname="MyClass" time="1.0">
    <failure message="assertion failed" type="AssertionError">Expected 1 but got 2</failure>
  </testcase>
</testsuite>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse should not return an error: %v", err)
	}

	if result.Tests != 2 {
		t.Errorf("expected 2 tests, got %d", result.Tests)
	}
	if result.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", result.Failures)
	}
	if len(result.TestSuites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(result.TestSuites))
	}

	suite := result.TestSuites[0]
	if len(suite.TestCases) != 2 {
		t.Errorf("expected 2 test cases, got %d", len(suite.TestCases))
	}
}

func TestParse_MultipleTestSuites(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="AllTests" tests="3" failures="1" errors="0">
  <testsuite name="Suite1" tests="2" failures="1" time="1.0">
    <testcase name="test1" time="0.5"/>
    <testcase name="test2" time="0.5">
      <failure message="failed">Error details</failure>
    </testcase>
  </testsuite>
  <testsuite name="Suite2" tests="1" failures="0" time="0.5">
    <testcase name="test3" time="0.5"/>
  </testsuite>
</testsuites>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse should not return an error: %v", err)
	}

	if result.Tests != 3 {
		t.Errorf("expected 3 tests, got %d", result.Tests)
	}
	if len(result.TestSuites) != 2 {
		t.Errorf("expected 2 test suites, got %d", len(result.TestSuites))
	}
}

func TestFormatOutput(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="MyTest" tests="2" failures="1" errors="0" time="1.5">
  <testcase name="test_success" classname="MyClass" time="0.5"/>
  <testcase name="test_failure" classname="MyClass" time="1.0">
    <failure message="assertion failed" type="AssertionError">Expected 1 but got 2</failure>
  </testcase>
</testsuite>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse should not return an error: %v", err)
	}

	output := FormatOutput(result)

	checks := []string{"Test Summary", "Failed Tests", "test_failure", "Passed Tests"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output should contain %q, got:\n%s", check, output)
		}
	}
}

func TestParse_ComputesStats(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="MyTest">
  <testcase name="test1" classname="MyClass" time="0.5"/>
  <testcase name="test2" classname="MyClass" time="1.0">
    <failure message="failed">Error</failure>
  </testcase>
  <testcase name="test3" classname="MyClass" time="0.3">
    <error message="error">Error details</error>
  </testcase>
  <testcase name="test4" classname="MyClass" time="0.2">
    <skipped/>
  </testcase>
</testsuite>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse should not return an error: %v", err)
	}

	if result.Tests != 4 {
		t.Errorf("expected 4 tests (computed), got %d", result.Tests)
	}
	if result.Failures != 1 {
		t.Errorf("expected 1 failure (computed), got %d", result.Failures)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error (computed), got %d", result.Errors)
	}

	expectedTime := 0.5 + 1.0 + 0.3 + 0.2
	if result.Time != expectedTime {
		t.Errorf("expected time %.2f, got %.2f", expectedTime, result.Time)
	}

	suite := result.TestSuites[0]
	if suite.Tests != 4 {
		t.Errorf("expected suite to have 4 tests (computed), got %d", suite.Tests)
	}
	if suite.Skipped != 1 {
		t.Errorf("expected suite to have 1 skipped (computed), got %d", suite.Skipped)
	}
}

func TestParse_EmptyTestSuites(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="EmptyRun" tests="0" failures="0" errors="0" time="0.0">
</testsuites>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse should not return an error for empty testsuites: %v", err)
	}

	if result.Name != "EmptyRun" {
		t.Errorf("expected name 'EmptyRun', got %q", result.Name)
	}
	if result.Tests != 0 {
		t.Errorf("expected 0 tests, got %d", result.Tests)
	}
	if result.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", result.Failures)
	}
	if len(result.TestSuites) != 0 {
		t.Errorf("expected 0 test suites, got %d", len(result.TestSuites))
	}
}

func TestToSummary(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="MyTest" tests="4" failures="1" errors="1" time="2.0">
  <testcase name="test1" classname="MyClass" time="0.5"/>
  <testcase name="test2" classname="MyClass" time="1.0">
    <failure message="failed">Error</failure>
  </testcase>
  <testcase name="test3" classname="MyClass" time="0.3">
    <error message="error">Error details</error>
  </testcase>
  <testcase name="test4" classname="MyClass" time="0.2">
    <skipped/>
  </testcase>
</testsuite>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	summary := result.ToSummary()
	if summary.Tests != 4 {
		t.Errorf("expected 4 tests, got %d", summary.Tests)
	}
	if summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", summary.Passed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summary.Skipped)
	}
	if len(summary.Suites) != 1 {
		t.Errorf("expected 1 suite, got %d", len(summary.Suites))
	}
}

func TestTestCaseStatuses(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="StatusTest">
  <testcase name="pass" time="0.1"/>
  <testcase name="fail" time="0.2">
    <failure message="bad"/>
  </testcase>
  <testcase name="err" time="0.3">
    <error message="oops"/>
  </testcase>
  <testcase name="skip" time="0.0">
    <skipped/>
  </testcase>
</testsuite>`

	result, err := Parse(xmlContent)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	expected := map[string]string{
		"pass": "PASSED",
		"fail": "FAILED",
		"err":  "ERROR",
		"skip": "SKIPPED",
	}

	for _, suite := range result.TestSuites {
		for _, tc := range suite.TestCases {
			want, ok := expected[tc.Name]
			if !ok {
				t.Errorf("unexpected test case: %s", tc.Name)
				continue
			}
			if tc.Status != want {
				t.Errorf("test %q: expected status %q, got %q", tc.Name, want, tc.Status)
			}
		}
	}
}
