package experiment

import (
	"errors"
	"github.com/lawrencewoodman/ddataset"
	"github.com/lawrencewoodman/ddataset/dcsv"
	"github.com/vlifesystems/rulehunter/aggregators"
	"github.com/vlifesystems/rulehunter/experiment"
	"github.com/vlifesystems/rulehunter/goal"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadExperiment(t *testing.T) {
	cases := []struct {
		filename       string
		wantExperiment *experiment.Experiment
		wantTags       []string
	}{
		{filepath.Join("fixtures", "flow.json"),
			&experiment.Experiment{
				Title: "What would indicate good flow?",
				Dataset: dcsv.New(
					filepath.Join("fixtures", "flow.csv"),
					true,
					rune(','),
					[]string{"group", "district", "height", "flow"},
				),
				ExcludeFieldNames: []string{"flow"},
				Aggregators: []aggregators.Aggregator{
					aggregators.MustNew("goodFlowAccuracy", "accuracy", "flow > 60"),
				},
				Goals: []*goal.Goal{goal.MustNew("goodFlowAccuracy > 10")},
				SortOrder: []experiment.SortField{
					experiment.SortField{"goodFlowAccuracy", experiment.DESCENDING},
					experiment.SortField{"numMatches", experiment.DESCENDING},
				},
			},
			[]string{"test", "fred / ned"},
		},
		{filepath.Join("fixtures", "debt.json"),
			&experiment.Experiment{
				Title: "What would predict people being helped to be debt free?",
				Dataset: dcsv.New(
					filepath.Join("..", "fixtures", "debt.csv"),
					true,
					rune(','),
					[]string{
						"name",
						"balance",
						"numCards",
						"martialStatus",
						"tertiaryEducated",
						"success",
					},
				),
				ExcludeFieldNames: []string{"success"},
				Aggregators: []aggregators.Aggregator{
					aggregators.MustNew("helpedAccuracy", "accuracy", "success"),
				},
				Goals: []*goal.Goal{goal.MustNew("helpedAccuracy > 10")},
				SortOrder: []experiment.SortField{
					experiment.SortField{"helpedAccuracy", experiment.DESCENDING},
					experiment.SortField{"numMatches", experiment.DESCENDING},
				},
			},
			[]string{"debt"},
		},
	}
	for _, c := range cases {
		gotExperiment, gotTags, err := loadExperiment(c.filename)
		if err != nil {
			t.Errorf("loadExperiment(%s) err: %s", c.filename, err)
			return
		}
		err = checkExperimentMatch(gotExperiment, c.wantExperiment)
		if err != nil {
			t.Errorf("loadExperiment(%s) experiments don't match: %s\ngotExperiment: %s, wantExperiment: %s",
				c.filename, err, gotExperiment, c.wantExperiment)
			return
		}
		if !reflect.DeepEqual(gotTags, c.wantTags) {
			t.Errorf("loadExperiment(%s) gotTags: %s, wantTags",
				c.filename, gotTags, c.wantTags)
			return
		}
	}
}

func TestLoadExperiment_error(t *testing.T) {
	cases := []struct {
		filename string
		wantErr  error
	}{
		{filepath.Join("fixtures", "flow_no_title.json"),
			errors.New("Experiment field missing: title")},
		{filepath.Join("fixtures", "flow_no_dataset.json"),
			errors.New("Experiment field missing: dataset")},
		{filepath.Join("fixtures", "flow_no_csv.json"),
			errors.New("Experiment field missing: csv")},
		{filepath.Join("fixtures", "flow_no_csv_filename.json"),
			errors.New("Experiment field missing: csv > filename")},
		{filepath.Join("fixtures", "flow_no_csv_separator.json"),
			errors.New("Experiment field missing: csv > separator")},
		{filepath.Join("fixtures", "flow_no_sql.json"),
			errors.New("Experiment field missing: sql")},
		{filepath.Join("fixtures", "flow_no_sql_drivername.json"),
			errors.New("Experiment field missing: sql > driverName")},
		{filepath.Join("fixtures", "flow_invalid_sql_drivername.json"),
			errors.New("Experiment has invalid sql > driverName")},
		{filepath.Join("fixtures", "flow_no_sql_datasourcename.json"),
			errors.New("Experiment field missing: sql > dataSourceName")},
		{filepath.Join("fixtures", "flow_no_sql_query.json"),
			errors.New("Experiment field missing: sql > query")},
	}
	for _, c := range cases {
		_, _, err := loadExperiment(c.filename)
		if err == nil {
			t.Errorf("loadExperiment(%s) no error, wantErr:%s",
				c.filename, c.wantErr)
			return
		}
		if err.Error() != c.wantErr.Error() {
			t.Errorf("loadExperiment(%s) gotErr: %s, wantErr:%s",
				c.filename, err, c.wantErr)
			return
		}
	}
}

/***********************
   Helper functions
************************/

func checkExperimentMatch(
	e1 *experiment.Experiment,
	e2 *experiment.Experiment,
) error {
	if e1.Title != e2.Title {
		return errors.New("Titles don't match")
	}
	if !areStringArraysEqual(e1.ExcludeFieldNames, e2.ExcludeFieldNames) {
		return errors.New("ExcludeFieldNames don't match")
	}
	if !areGoalExpressionsEqual(e1.Goals, e2.Goals) {
		return errors.New("Goals don't match")
	}
	if !areAggregatorsEqual(e1.Aggregators, e2.Aggregators) {
		return errors.New("Aggregators don't match")
	}
	if !areSortOrdersEqual(e1.SortOrder, e2.SortOrder) {
		return errors.New("Sort Orders don't match")
	}
	return checkDatasetsEqual(e1.Dataset, e2.Dataset)
}

func checkDatasetsEqual(ds1, ds2 ddataset.Dataset) error {
	conn1, err := ds1.Open()
	if err != nil {
		return err
	}
	conn2, err := ds2.Open()
	if err != nil {
		return err
	}
	for {
		conn1Next := conn1.Next()
		conn2Next := conn2.Next()
		if conn1Next != conn2Next {
			errors.New("Datasets don't finish at same point")
		}
		if !conn1Next {
			break
		}

		conn1Record := conn1.Read()
		conn2Record := conn2.Read()
		if !reflect.DeepEqual(conn1Record, conn2Record) {
			return errors.New("Datasets don't match")
		}
	}
	if conn1.Err() != conn2.Err() {
		return errors.New("Datasets final error doesn't match")
	}
	return nil
}

func areStringArraysEqual(a1 []string, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i, e := range a1 {
		if e != a2[i] {
			return false
		}
	}
	return true
}

func areGoalExpressionsEqual(g1 []*goal.Goal, g2 []*goal.Goal) bool {
	if len(g1) != len(g2) {
		return false
	}
	for i, g := range g1 {
		if g.String() != g2[i].String() {
			return false
		}
	}
	return true

}

func areAggregatorsEqual(
	a1 []aggregators.Aggregator,
	a2 []aggregators.Aggregator,
) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i, e := range a1 {
		if !e.IsEqual(a2[i]) {
			return false
		}
	}
	return true
}

func areSortOrdersEqual(
	so1 []experiment.SortField,
	so2 []experiment.SortField,
) bool {
	if len(so1) != len(so2) {
		return false
	}
	for i, sf1 := range so1 {
		sf2 := so2[i]
		if sf1.Field != sf2.Field || sf1.Direction != sf2.Direction {
			return false
		}
	}
	return true
}