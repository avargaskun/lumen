package tasks

import "testing"

func TestParseTestList_JSONArray(t *testing.T) {
	input := `["test_one", "test_two", "test_three"]`
	got := parseTestList(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	if got[0] != "test_one" || got[1] != "test_two" || got[2] != "test_three" {
		t.Errorf("unexpected items: %v", got)
	}
}

func TestParseTestList_PythonRepr(t *testing.T) {
	input := `['test_has_key_race', 'test_cache_clear']`
	got := parseTestList(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0] != "test_has_key_race" {
		t.Errorf("got[0] = %q, want %q", got[0], "test_has_key_race")
	}
	if got[1] != "test_cache_clear" {
		t.Errorf("got[1] = %q, want %q", got[1], "test_cache_clear")
	}
}

func TestParseTestList_Empty(t *testing.T) {
	for _, input := range []string{"", "[]", "  "} {
		got := parseTestList(input)
		if len(got) != 0 {
			t.Errorf("parseTestList(%q): expected empty, got %v", input, got)
		}
	}
}

func TestParseTestList_SingleItem(t *testing.T) {
	got := parseTestList(`["single_test"]`)
	if len(got) != 1 || got[0] != "single_test" {
		t.Errorf("expected [single_test], got %v", got)
	}
}

func TestDetectTestCmd(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"django/django", "python tests/runtests.py --verbosity 2 --settings test_sqlite --parallel 1"},
		{"sympy/sympy", "bin/test -C --verbose"},
		{"sphinx-doc/sphinx", "python -m pytest --no-header -rN"},
		{"pallets/flask", "python -m pytest --no-header -rN"},
		{"psf/requests", "python -m pytest --no-header -rN"},
		{"pylint-dev/pylint", "python -m pytest --no-header -rN"},
		{"matplotlib/matplotlib", "python -m pytest --no-header -rN"},
	}

	for _, tc := range tests {
		got := detectTestCmd(tc.repo)
		if got != tc.want {
			t.Errorf("detectTestCmd(%q) = %q, want %q", tc.repo, got, tc.want)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"django/django", "python"},
		{"sympy/sympy", "python"},
		{"pallets/flask", "python"},
		{"unknown/repo", "python"},
	}

	for _, tc := range tests {
		got := detectLanguage(tc.repo)
		if got != tc.want {
			t.Errorf("detectLanguage(%q) = %q, want %q", tc.repo, got, tc.want)
		}
	}
}

func TestConvertSWEBenchRow_Valid(t *testing.T) {
	row := sweBenchRow{
		InstanceID:       "django__django-16379",
		Repo:             "django/django",
		BaseCommit:       "abc123",
		ProblemStatement: "Fix the cache race",
		FailToPassStr:    `["test_has_key_race"]`,
		PassToPassStr:    `["test_simple"]`,
	}

	task := convertSWEBenchRow(row)
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.ID != "django__django-16379" {
		t.Errorf("ID = %q", task.ID)
	}
	if task.Repo != "https://github.com/django/django" {
		t.Errorf("Repo = %q", task.Repo)
	}
	if task.Language != "python" {
		t.Errorf("Language = %q", task.Language)
	}
	if task.Validation.TestCmd != "python tests/runtests.py --verbosity 2 --settings test_sqlite --parallel 1" {
		t.Errorf("TestCmd = %q", task.Validation.TestCmd)
	}
	if len(task.Validation.FailToPass) != 1 || task.Validation.FailToPass[0] != "test_has_key_race" {
		t.Errorf("FailToPass = %v", task.Validation.FailToPass)
	}
	if len(task.Validation.PassToPass) != 1 || task.Validation.PassToPass[0] != "test_simple" {
		t.Errorf("PassToPass = %v", task.Validation.PassToPass)
	}
}

func TestConvertSWEBenchRow_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		row  sweBenchRow
	}{
		{"empty ID", sweBenchRow{Repo: "r", BaseCommit: "c", FailToPassStr: `["t"]`}},
		{"empty repo", sweBenchRow{InstanceID: "i", BaseCommit: "c", FailToPassStr: `["t"]`}},
		{"empty commit", sweBenchRow{InstanceID: "i", Repo: "r", FailToPassStr: `["t"]`}},
		{"empty fail_to_pass", sweBenchRow{InstanceID: "i", Repo: "r", BaseCommit: "c"}},
	}

	for _, tc := range tests {
		task := convertSWEBenchRow(tc.row)
		if task != nil {
			t.Errorf("%s: expected nil task, got %+v", tc.name, task)
		}
	}
}

func TestValidateTask_Defaults(t *testing.T) {
	task := Task{
		ID:               "test",
		Repo:             "https://github.com/example/repo",
		BaseCommit:       "abc",
		ProblemStatement: "fix it",
		Validation: Validation{
			TestCmd:    "pytest",
			FailToPass: []string{"test"},
		},
	}

	if err := validateTask(&task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.MaxTurns != 30 {
		t.Errorf("MaxTurns: got %d, want 30", task.MaxTurns)
	}
	if task.MaxBudgetUSD != 2.0 {
		t.Errorf("MaxBudgetUSD: got %f, want 2.0", task.MaxBudgetUSD)
	}
}
