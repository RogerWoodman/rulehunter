/*
	Copyright (C) 2016 vLife Systems Ltd <http://vlifesystems.com>
	This file is part of Rulehunter.

	Rulehunter is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	Rulehunter is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with Rulehunter; see the file COPYING.  If not, see
	<http://www.gnu.org/licenses/>.
*/

package rulehunter

import (
	"errors"
	"fmt"
	"github.com/lawrencewoodman/dexpr"
	"github.com/lawrencewoodman/dlit"
	"github.com/vlifesystems/rulehunter/internal"
	"regexp"
	"sort"
	"strings"
)

type ruleGeneratorFunc func(
	*Description,
	[]string,
	string,
) ([]*Rule, error)

func GenerateRules(
	inputDescription *Description,
	excludeFields []string,
) ([]*Rule, error) {
	rules := make([]*Rule, 1)
	ruleGenerators := []ruleGeneratorFunc{
		generateIntRules, generateFloatRules, generateStringRules,
		generateCompareNumericRules, generateCompareStringRules,
		generateInNiRules,
	}
	rules[0] = mustNewRule("true()")
	for field, _ := range inputDescription.fields {
		if !stringInSlice(field, excludeFields) {
			for _, ruleGenerator := range ruleGenerators {
				newRules, err := ruleGenerator(inputDescription, excludeFields, field)
				if err != nil {
					return nil, err
				}
				rules = append(rules, newRules...)
			}
		}
	}
	return rules, nil
}

func CombineRules(rules []*Rule) []*Rule {
	combinedRules := make([]*Rule, 0)
	numRules := len(rules)
	for i := 0; i < numRules-1; i++ {
		for j := i + 1; j < numRules; j++ {
			ruleI := rules[i].String()
			ruleJ := rules[j].String()
			if ruleI != "true()" && ruleJ != "true()" {
				andRuleStr := fmt.Sprintf("%s && %s", ruleI, ruleJ)
				orRuleStr := fmt.Sprintf("%s || %s", ruleI, ruleJ)
				andRule := mustNewRule(andRuleStr)
				orRule := mustNewRule(orRuleStr)
				combinedRules = append(combinedRules, andRule)
				combinedRules = append(combinedRules, orRule)
			}
		}
	}
	return combinedRules
}

func stringInSlice(s string, strings []string) bool {
	for _, x := range strings {
		if x == s {
			return true
		}
	}
	return false
}

func generateIntRules(
	inputDescription *Description,
	excludeFields []string,
	field string,
) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	if fd.kind != internal.INT {
		return []*Rule{}, nil
	}
	rulesMap := make(map[string]*Rule)
	min, _ := fd.min.Int()
	max, _ := fd.max.Int()
	values := fd.values
	diff := max - min
	step := diff / 10
	if step == 0 {
		step = 1
	}
	// i set to 0 to make more tweakable
	for i := int64(0); i < diff; i += step {
		n := min + i
		exprStr := fmt.Sprintf("%s >= %d", field, n)
		rule, err := newRule(exprStr)
		if err != nil {
			return nil, err
		}
		rulesMap[exprStr] = rule
	}

	for i := step; i <= diff; i += step {
		n := min + i
		exprStr := fmt.Sprintf("%s <= %d", field, n)
		rule, err := newRule(exprStr)
		if err != nil {
			return nil, err
		}
		rulesMap[exprStr] = rule
	}

	if len(values) >= 2 {
		for _, v := range values {
			n, isInt := v.Int()
			if !isInt {
				return nil, errors.New(fmt.Sprintf("value isn't int: %s", v))
			}
			exprStr := fmt.Sprintf("%s == %d", field, n)
			_rule, err := newRule(exprStr)
			if err != nil {
				return nil, err
			}
			rulesMap[exprStr] = _rule

			exprStr = fmt.Sprintf("%s != %d", field, n)
			_rule, err = newRule(exprStr)
			if err != nil {
				return nil, err
			}
			rulesMap[exprStr] = _rule
		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

var matchTrailingZerosRegexp = regexp.MustCompile("^(\\d+.[^ 0])(0+)$")
var matchAllZerosRegexp = regexp.MustCompile("^(0)(.0+)$")

func floatToString(f float64, maxDP int) string {
	fStr := fmt.Sprintf("%.*f", maxDP, f)
	fStr = matchTrailingZerosRegexp.ReplaceAllString(fStr, "$1")
	return matchAllZerosRegexp.ReplaceAllString(fStr, "0")
}

// TODO: For each rule give all dp numbers 0..maxDP
func generateFloatRules(
	inputDescription *Description,
	excludeFields []string, field string) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	if fd.kind != internal.FLOAT {
		return []*Rule{}, nil
	}
	rulesMap := make(map[string]*Rule)
	min, _ := fd.min.Float()
	max, _ := fd.max.Float()
	maxDP := fd.maxDP
	values := fd.values
	diff := max - min
	step := diff / 10.0

	// i set to 0 to make more tweakable
	for i := float64(0); i < diff; i += step {
		n := min + i
		nStr := floatToString(n, maxDP)
		exprStr := fmt.Sprintf("%s >= %s", field, nStr)
		rule, err := newRule(exprStr)
		if err != nil {
			return nil, err
		}
		rulesMap[exprStr] = rule
	}

	for i := step; i <= diff; i += step {
		n := min + i
		nStr := floatToString(n, maxDP)
		exprStr := fmt.Sprintf("%s <= %s", field, nStr)
		rule, err := newRule(exprStr)
		if err != nil {
			return nil, err
		}
		rulesMap[exprStr] = rule
	}

	if len(values) >= 2 {
		for _, v := range values {
			n, isFloat := v.Float()
			if !isFloat {
				return nil, errors.New(fmt.Sprintf("value isn't float: %s", v))
			}
			nStr := floatToString(n, maxDP)
			exprStr := fmt.Sprintf("%s == %s", field, nStr)
			_rule, err := newRule(exprStr)
			if err != nil {
				return nil, err
			}
			rulesMap[exprStr] = _rule

			exprStr = fmt.Sprintf("%s != %s", field, nStr)
			_rule, err = newRule(exprStr)
			if err != nil {
				return nil, err
			}
			rulesMap[exprStr] = _rule
		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

func generateCompareNumericRules(
	inputDescription *Description,
	excludeFields []string,
	field string,
) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	if fd.kind != internal.INT && fd.kind != internal.FLOAT {
		return []*Rule{}, nil
	}
	fieldNum := calcFieldNum(inputDescription.fields, field)
	rulesMap := make(map[string]*Rule)
	exprFmts := []string{
		"%s < %s", "%s <= %s", "%s == %s", "%s != %s", "%s >= %s", "%s > %s",
	}

	for oField, oFd := range inputDescription.fields {
		oFieldNum := calcFieldNum(inputDescription.fields, oField)
		isComparable := hasComparableNumberRange(fd, oFd)
		if fieldNum < oFieldNum &&
			isComparable &&
			!stringInSlice(oField, excludeFields) {
			for _, exprFmt := range exprFmts {
				exprStr := fmt.Sprintf(exprFmt, field, oField)
				rule, err := newRule(exprStr)
				if err != nil {
					return nil, err
				}
				rulesMap[exprStr] = rule
			}
		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

func generateCompareStringRules(
	inputDescription *Description,
	excludeFields []string,
	field string,
) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	if fd.kind != internal.STRING {
		return []*Rule{}, nil
	}
	fieldNum := calcFieldNum(inputDescription.fields, field)
	rulesMap := make(map[string]*Rule)
	exprFmts := []string{
		"%s == %s", "%s != %s",
	}
	for oField, oFd := range inputDescription.fields {
		if oFd.kind == internal.STRING {
			oFieldNum := calcFieldNum(inputDescription.fields, oField)
			numSharedValues := calcNumSharedValues(fd, oFd)
			if fieldNum < oFieldNum &&
				numSharedValues >= 2 &&
				!stringInSlice(oField, excludeFields) {
				for _, exprFmt := range exprFmts {
					exprStr := fmt.Sprintf(exprFmt, field, oField)
					rule, err := newRule(exprStr)
					if err != nil {
						return nil, err
					}
					rulesMap[exprStr] = rule
				}
			}
		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

func calcNumSharedValues(
	fd1 *fieldDescription,
	fd2 *fieldDescription,
) int {
	numShared := 0
	for _, v1 := range fd1.values {
		for _, v2 := range fd2.values {
			if v1.String() == v2.String() {
				numShared++
			}
		}
	}
	return numShared
}

func generateStringRules(
	inputDescription *Description,
	excludeFields []string,
	field string,
) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	if fd.kind != internal.STRING {
		return []*Rule{}, nil
	}
	rulesMap := make(map[string]*Rule)

	for _, v := range fd.values {
		s := v.String()
		exprStr := fmt.Sprintf("%s == \"%s\"", field, s)
		_rule, err := newRule(exprStr)
		if err != nil {
			return nil, err
		}
		rulesMap[exprStr] = _rule
		if len(fd.values) > 2 {
			exprStr := fmt.Sprintf("%s != \"%s\"", field, s)
			_rule, err := newRule(exprStr)
			if err != nil {
				return nil, err
			}
			rulesMap[exprStr] = _rule
		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

func isNumberField(fd *fieldDescription) bool {
	return fd.kind == internal.INT || fd.kind == internal.FLOAT
}

func hasComparableNumberRange(
	fd1 *fieldDescription,
	fd2 *fieldDescription,
) bool {
	if !isNumberField(fd1) || !isNumberField(fd2) {
		return false
	}
	var isComparable bool
	vars := map[string]*dlit.Literal{
		"min1": fd1.min,
		"max1": fd1.max,
		"min2": fd2.min,
		"max2": fd2.max,
	}
	funcs := map[string]dexpr.CallFun{}
	expr, err := dexpr.New("min1 < max2 && max1 > min2")
	if err != nil {
		return false
	}
	isComparable, err = expr.EvalBool(vars, funcs)
	if err != nil {
		return false
	}
	return isComparable
}

func rulesMapToArray(rulesMap map[string]*Rule) []*Rule {
	rules := make([]*Rule, len(rulesMap))
	i := 0
	for _, expr := range rulesMap {
		rules[i] = expr
		i++
	}
	return rules
}

func generateInNiRules(
	inputDescription *Description,
	excludeFields []string,
	field string,
) ([]*Rule, error) {
	fd := inputDescription.fields[field]
	numValues := len(fd.values)
	if fd.kind != internal.STRING &&
		fd.kind != internal.FLOAT &&
		fd.kind != internal.INT ||
		numValues <= 3 || numValues > 12 {
		return []*Rule{}, nil
	}
	rulesMap := make(map[string]*Rule)
	exprFmts := []string{
		"in(%s,%s)", "ni(%s,%s)",
	}
	for i := 3; ; i++ {
		numOnBits := calcNumOnBits(i)
		if numOnBits >= numValues {
			break
		}
		if numOnBits >= 2 && numOnBits <= 5 && numOnBits < (numValues-1) {
			compareValuesStr := makeCompareValuesStr(fd.values, i)
			for _, exprFmt := range exprFmts {
				exprStr := fmt.Sprintf(exprFmt, field, compareValuesStr)
				rule, err := newRule(exprStr)
				if err != nil {
					return nil, err
				}
				rulesMap[exprStr] = rule
			}

		}
	}
	rules := rulesMapToArray(rulesMap)
	return rules, nil
}

func makeCompareValues(values []*dlit.Literal, i int) []*dlit.Literal {
	bStr := fmt.Sprintf("%b", i)
	numOnBits := calcNumOnBits(i)
	numValues := len(values)
	j := numValues - 1
	compareValues := make([]*dlit.Literal, numOnBits)
	k := 0
	for _, b := range reverseString(bStr) {
		if b == '1' {
			compareValues[k] = values[numValues-1-j]
			k++
		}
		j -= 1
	}
	return compareValues
}

func reverseString(s string) (r string) {
	for _, v := range s {
		r = string(v) + r
	}
	return
}

func makeCompareValuesStr(values []*dlit.Literal, i int) string {
	compareValues := makeCompareValues(values, i)
	str := fmt.Sprintf("\"%s\"", compareValues[0].String())
	for _, v := range compareValues[1:] {
		str += fmt.Sprintf(",\"%s\"", v)
	}
	return str
}

func calcNumOnBits(i int) int {
	bStr := fmt.Sprintf("%b", i)
	return strings.Count(bStr, "1")
}

func calcFieldNum(
	fieldDescriptions map[string]*fieldDescription,
	fieldN string,
) int {
	fields := make([]string, len(fieldDescriptions))
	i := 0
	for field, _ := range fieldDescriptions {
		fields[i] = field
		i++
	}
	sort.Strings(fields)
	j := 0
	for _, field := range fields {
		if field == fieldN {
			return j
		}
		j++
	}
	panic("Can't field field in fieldDescriptions")
}
