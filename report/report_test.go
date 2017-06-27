package report

import (
	"errors"
	"fmt"
	"github.com/lawrencewoodman/dlit"
	"github.com/vlifesystems/rhkit"
	"github.com/vlifesystems/rhkit/aggregators"
	"github.com/vlifesystems/rhkit/experiment"
	"github.com/vlifesystems/rhkit/rule"
	"github.com/vlifesystems/rulehunter/config"
	"github.com/vlifesystems/rulehunter/internal/testhelpers"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	assessment := rhkit.NewAssessment(20)
	assessment.RuleAssessments = []*rhkit.RuleAssessment{
		&rhkit.RuleAssessment{
			Rule: rule.NewEQFV("month", dlit.NewString("may")),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("2142"),
				"percentMatches": dlit.MustNew("242"),
				"numIncomeGt2":   dlit.MustNew("22"),
				"goalsScore":     dlit.MustNew(20.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", false},
			},
		},
		&rhkit.RuleAssessment{
			Rule: rule.NewGEFV("rate", dlit.MustNew(789.2)),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("3142"),
				"percentMatches": dlit.MustNew("342"),
				"numIncomeGt2":   dlit.MustNew("32"),
				"goalsScore":     dlit.MustNew(30.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", false},
			},
		},
		&rhkit.RuleAssessment{
			Rule: rule.NewTrue(),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("142"),
				"percentMatches": dlit.MustNew("42"),
				"numIncomeGt2":   dlit.MustNew("2"),
				"goalsScore":     dlit.MustNew(0.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
			},
		},
	}
	experiment := &experiment.Experiment{
		Title: "some title",
		SortOrder: []experiment.SortField{
			experiment.SortField{
				Field:     "goalsScore",
				Direction: experiment.DESCENDING,
			},
			experiment.SortField{
				Field:     "percentMatches",
				Direction: experiment.ASCENDING,
			},
		},
		Aggregators: []aggregators.AggregatorSpec{
			aggregators.MustNew("numMatches", "count", "true()"),
			aggregators.MustNew(
				"percentMatches",
				"calc",
				"roundto(100.0 * numMatches / numRecords, 2)",
			),
			aggregators.MustNew("numIncomeGt2", "count", "income > 2"),
			aggregators.MustNew("goalsScore", "goalsscore"),
		},
	}
	experimentFilename := "somename.yaml"
	tags := []string{"bank", "test / fred"}
	wantAggregatorDescs := []AggregatorDesc{
		AggregatorDesc{Name: "numMatches", Kind: "count", Arg: "true()"},
		AggregatorDesc{
			Name: "percentMatches",
			Kind: "calc",
			Arg:  "roundto(100.0 * numMatches / numRecords, 2)",
		},
		AggregatorDesc{Name: "numIncomeGt2", Kind: "count", Arg: "income > 2"},
		AggregatorDesc{Name: "goalsScore", Kind: "goalsscore", Arg: ""},
	}
	wantAssessments := []*Assessment{
		&Assessment{
			Rule: "rate >= 789.2",
			Aggregators: []*Aggregator{
				&Aggregator{Name: "goalsScore", Value: "30.1", Difference: "30"},
				&Aggregator{Name: "numIncomeGt2", Value: "32", Difference: "30"},
				&Aggregator{Name: "numMatches", Value: "3142", Difference: "3000"},
				&Aggregator{Name: "percentMatches", Value: "342", Difference: "300"},
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", false},
			},
		},
		&Assessment{
			Rule: "month == \"may\"",
			Aggregators: []*Aggregator{
				&Aggregator{Name: "goalsScore", Value: "20.1", Difference: "20"},
				&Aggregator{Name: "numIncomeGt2", Value: "22", Difference: "20"},
				&Aggregator{Name: "numMatches", Value: "2142", Difference: "2000"},
				&Aggregator{Name: "percentMatches", Value: "242", Difference: "200"},
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", false},
			},
		},
		&Assessment{
			Rule: "true()",
			Aggregators: []*Aggregator{
				&Aggregator{Name: "goalsScore", Value: "0.1", Difference: "0"},
				&Aggregator{Name: "numIncomeGt2", Value: "2", Difference: "0"},
				&Aggregator{Name: "numMatches", Value: "142", Difference: "0"},
				&Aggregator{Name: "percentMatches", Value: "42", Difference: "0"},
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
			},
		},
	}
	report := New(assessment, experiment, experimentFilename, tags)
	if report.Title != experiment.Title {
		t.Errorf("New report.Title got: %s, want: %s",
			report.Title, experiment.Title)
	}
	if !reflect.DeepEqual(report.Tags, tags) {
		t.Errorf("New report.Tags got: %s, want: %s",
			report.Tags, tags)
	}
	if time.Now().Sub(report.Stamp).Seconds() > 1 {
		t.Errorf("New report.Stamp got: %s, want: %s",
			report.Stamp, time.Now())
	}
	if report.ExperimentFilename != experimentFilename {
		t.Errorf("New report.ExperimentFilename got: %s, want: %s",
			report.ExperimentFilename, experimentFilename)
	}
	if report.NumRecords != assessment.NumRecords {
		t.Errorf("New report.NumRecords got: %s, want: %s",
			report.NumRecords, assessment.NumRecords)
	}
	if !reflect.DeepEqual(report.SortOrder, experiment.SortOrder) {
		t.Errorf("New report.SortOrder got: %s, want: %s",
			report.SortOrder, experiment.SortOrder)
	}
	if !reflect.DeepEqual(report.Aggregators, wantAggregatorDescs) {
		t.Errorf("New report.Aggregators got: %s, want: %s",
			report.Aggregators, wantAggregatorDescs)
	}
	err := checkAssessmentsMatch(report.Assessments, wantAssessments)
	if err != nil {
		t.Errorf("New report.Assessments don't match: %s", err)
	}
}

func TestWriteLoadJSON(t *testing.T) {
	// File mode permission used as standard for the html content:
	// No special permission bits
	// User: Read, Write Execute
	// Group: Read
	// Other: None
	const modePerm = 0740

	tmpDir := testhelpers.TempDir(t)
	defer os.RemoveAll(tmpDir)
	reportsDir := filepath.Join(tmpDir, "reports")
	if err := os.MkdirAll(reportsDir, modePerm); err != nil {
		t.Fatalf("MkdirAll: %s", err)
	}
	assessment := rhkit.NewAssessment(20)
	assessment.RuleAssessments = []*rhkit.RuleAssessment{
		&rhkit.RuleAssessment{
			Rule: rule.NewEQFV("month", dlit.NewString("may")),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("2142"),
				"percentMatches": dlit.MustNew("242"),
				"numIncomeGt2":   dlit.MustNew("22"),
				"goalsScore":     dlit.MustNew(20.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
			},
		},
		&rhkit.RuleAssessment{
			Rule: rule.NewGEFV("rate", dlit.MustNew(789.2)),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("3142"),
				"percentMatches": dlit.MustNew("342"),
				"numIncomeGt2":   dlit.MustNew("32"),
				"goalsScore":     dlit.MustNew(30.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
			},
		},
		&rhkit.RuleAssessment{
			Rule: rule.NewTrue(),
			Aggregators: map[string]*dlit.Literal{
				"numMatches":     dlit.MustNew("142"),
				"percentMatches": dlit.MustNew("42"),
				"numIncomeGt2":   dlit.MustNew("2"),
				"goalsScore":     dlit.MustNew(0.1),
			},
			Goals: []*rhkit.GoalAssessment{
				&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
				&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
			},
		},
	}
	experiment := &experiment.Experiment{
		Title: "some title",
		SortOrder: []experiment.SortField{
			experiment.SortField{
				Field:     "goalsScore",
				Direction: experiment.DESCENDING,
			},
			experiment.SortField{
				Field:     "percentMatches",
				Direction: experiment.ASCENDING,
			},
		},
		Aggregators: []aggregators.AggregatorSpec{
			aggregators.MustNew("numMatches", "count", "true()"),
			aggregators.MustNew(
				"percentMatches",
				"calc",
				"roundto(100.0 * numMatches / numRecords, 2)",
			),
			aggregators.MustNew("numIncomeGt2", "count", "income > 2"),
			aggregators.MustNew("goalsScore", "goalsscore"),
		},
	}
	experimentFilename := "somename.yaml"
	tags := []string{"bank", "test / fred"}
	config := &config.Config{BuildDir: tmpDir}
	report := New(assessment, experiment, experimentFilename, tags)

	if err := report.WriteJSON(config); err != nil {
		t.Fatalf("WriteJSON: %s", err)
	}
	loadedReport, err := LoadJSON(config, experimentFilename)
	if err != nil {
		t.Fatalf("LoadJSON: %s", err)
	}
	if err := checkReportsMatch(report, loadedReport); err != nil {
		t.Errorf("Reports don't match: %s", err)
	}
}

func TestCalcTrueAggregatorDiff(t *testing.T) {
	trueAggregators := map[string]*dlit.Literal{
		"numMatches": dlit.MustNew(176),
		"profit":     dlit.MustNew(23),
		"bigNum":     dlit.MustNew(int64(math.MaxInt64)),
	}
	cases := []struct {
		name  string
		value *dlit.Literal
		want  string
	}{
		{name: "numMatches", value: dlit.MustNew(192), want: "16"},
		{name: "numMatches", value: dlit.MustNew(165), want: "-11"},
		{name: "bigNum",
			value: dlit.MustNew(int64(math.MinInt64)),
			want: dlit.MustNew(
				float64(math.MinInt64) - float64(math.MaxInt64),
			).String(),
		},
		{name: "bigNum",
			value: dlit.MustNew(errors.New("some error")),
			want:  "N/A",
		},
	}

	for _, c := range cases {
		got := calcTrueAggregatorDiff(trueAggregators, c.name, c.value)
		if got != c.want {
			t.Errorf("calcTrueAggregatorDifference(trueAggregators, %v, %v) got: %s, want: %s",
				c.name, c.value, got, c.want)
		}
	}
}

func TestGetSortedAggregatorNames(t *testing.T) {
	aggregators := map[string]*dlit.Literal{
		"numMatches": dlit.MustNew(176),
		"profit":     dlit.MustNew(23),
		"bigNum":     dlit.MustNew(int64(math.MaxInt64)),
	}
	want := []string{"bigNum", "numMatches", "profit"}
	got := getSortedAggregatorNames(aggregators)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getSortedAggregatorNames - got: %v, want: %v", got, want)
	}
}

func TestGetTrueAggregators(t *testing.T) {
	assessment := &rhkit.Assessment{
		NumRecords: 20,
		RuleAssessments: []*rhkit.RuleAssessment{
			&rhkit.RuleAssessment{
				Rule: rule.NewEQFV("month", dlit.NewString("may")),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("2142"),
					"percentMatches": dlit.MustNew("242"),
					"numIncomeGt2":   dlit.MustNew("22"),
					"goalsScore":     dlit.MustNew(20.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
			&rhkit.RuleAssessment{
				Rule: rule.NewGEFV("rate", dlit.MustNew(789.2)),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("3142"),
					"percentMatches": dlit.MustNew("342"),
					"numIncomeGt2":   dlit.MustNew("32"),
					"goalsScore":     dlit.MustNew(30.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
			&rhkit.RuleAssessment{
				Rule: rule.NewTrue(),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("142"),
					"percentMatches": dlit.MustNew("42"),
					"numIncomeGt2":   dlit.MustNew("2"),
					"goalsScore":     dlit.MustNew(0.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
		},
	}
	want := map[string]*dlit.Literal{
		"numMatches":     dlit.MustNew("142"),
		"percentMatches": dlit.MustNew("42"),
		"numIncomeGt2":   dlit.MustNew("2"),
		"goalsScore":     dlit.MustNew(0.1),
	}

	got, err := getTrueAggregators(assessment)
	if err != nil {
		t.Fatalf("getTrueAggregators: %s", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getTrueAggregators - got: %v, want: %v", got, want)
	}
}

func TestGetTrueAggregators_error(t *testing.T) {
	assessment := &rhkit.Assessment{
		NumRecords: 20,
		RuleAssessments: []*rhkit.RuleAssessment{
			&rhkit.RuleAssessment{
				Rule: rule.NewEQFV("month", dlit.NewString("may")),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("2142"),
					"percentMatches": dlit.MustNew("242"),
					"numIncomeGt2":   dlit.MustNew("22"),
					"goalsScore":     dlit.MustNew(20.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
			&rhkit.RuleAssessment{
				Rule: rule.NewTrue(),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("142"),
					"percentMatches": dlit.MustNew("42"),
					"numIncomeGt2":   dlit.MustNew("2"),
					"goalsScore":     dlit.MustNew(0.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
			&rhkit.RuleAssessment{
				Rule: rule.NewGEFV("rate", dlit.MustNew(789.2)),
				Aggregators: map[string]*dlit.Literal{
					"numMatches":     dlit.MustNew("3142"),
					"percentMatches": dlit.MustNew("342"),
					"numIncomeGt2":   dlit.MustNew("32"),
					"goalsScore":     dlit.MustNew(30.1),
				},
				Goals: []*rhkit.GoalAssessment{
					&rhkit.GoalAssessment{"numIncomeGt2 == 1", false},
					&rhkit.GoalAssessment{"numIncomeGt2 == 2", true},
				},
			},
		},
	}
	wantErr := errors.New("can't find true() rule")

	_, err := getTrueAggregators(assessment)
	if err == nil || err.Error() != wantErr.Error() {
		t.Errorf("getTrueAggregators: err: %s, wantErr: %s", err, wantErr)
	}
}

/******************************
 *  Helper Functions
 ******************************/

func checkReportsMatch(r1, r2 *Report) error {
	if r1.Title != r2.Title {
		return fmt.Errorf("Titles don't match - %s != %s", r1.Title, r2.Title)
	}
	if !reflect.DeepEqual(r1.Tags, r2.Tags) {
		return fmt.Errorf("Tags don't match - %v != %v", r1.Tags, r2.Tags)
	}
	if math.Abs(r1.Stamp.Sub(r2.Stamp).Seconds()) > 1 {
		return fmt.Errorf("Stamps don't match - %s != %s", r1.Stamp, r2.Stamp)
	}
	if r1.ExperimentFilename != r2.ExperimentFilename {
		return fmt.Errorf("ExperimentFilenames don't match - %s != %s",
			r1.ExperimentFilename, r2.ExperimentFilename)
	}
	if r1.NumRecords != r2.NumRecords {
		return fmt.Errorf("NumRecords don't match - %d != %d",
			r1.NumRecords, r2.NumRecords)
	}
	if !reflect.DeepEqual(r1.SortOrder, r2.SortOrder) {
		return fmt.Errorf("SortOrder don't match - %v != %v",
			r1.SortOrder, r2.SortOrder)
	}
	if !reflect.DeepEqual(r1.Aggregators, r2.Aggregators) {
		return fmt.Errorf("Aggregators don't match - %v != %v",
			r1.Aggregators, r2.Aggregators)
	}
	if !reflect.DeepEqual(r1.Assessments, r2.Assessments) {
		return fmt.Errorf("Assessments don't match - %v != %v",
			r1.Assessments, r2.Assessments)
	}
	return nil
}

func checkAssessmentsMatch(as1, as2 []*Assessment) error {
	if len(as1) != len(as2) {
		return fmt.Errorf("number of Assessments don't match %d != %d",
			len(as1), len(as2))
	}
	for i, assessment1 := range as1 {
		if assessment1.Rule != as2[i].Rule {
			return fmt.Errorf("assessment[%d] Rules don't match: %s != %s",
				i, assessment1.Rule, as2[i].Rule)
		}
		if !reflect.DeepEqual(assessment1.Aggregators, as2[i].Aggregators) {
			return fmt.Errorf("assessment[%d] Aggregators don't match: %s != %s",
				i, assessment1.Aggregators, as2[i].Aggregators)
		}
		if !reflect.DeepEqual(assessment1.Goals, as2[i].Goals) {
			return fmt.Errorf("assessment[%d] Goals don't match: %s != %s",
				i, assessment1.Goals, as2[i].Goals)
		}
	}
	return nil
}
