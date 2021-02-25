package controller

import (
	"reflect"
	"sort"
	"testing"
	"time"

	cronHPAV1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
)

func TestGetRecentUnmetScheduleTimes(t *testing.T) {
	// schedule is hourly on the hour
	schedule := "0 * * * ?"
	// T1 is a scheduled start time of that schedule
	T1, err := time.Parse(time.RFC3339, "2016-05-19T10:00:00Z")
	if err != nil {
		t.Errorf("test setup error: %v", err)
	}
	// T2 is a scheduled start time of that schedule after T1
	T2, err := time.Parse(time.RFC3339, "2016-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("test setup error: %v", err)
	}

	testCases := []struct {
		name                    string
		nowTime                 time.Time
		earliestTime            time.Time
		cron                    string
		startingDeadlineSeconds int32
		expectErr               bool
		expectResult            []time.Time
	}{
		{
			// Case 1: no known start times, and none needed yet.
			name:         "Case 1: no known start times, and none needed yet",
			nowTime:      T1.Add(-7 * time.Minute),
			earliestTime: T1.Add(-10 * time.Minute),
			cron:         schedule,
			expectErr:    false,
			expectResult: []time.Time{},
		},
		{
			// Case 2: no known start times, and one needed.
			name:         "Case 2: no known start times, and one needed",
			nowTime:      T1.Add(2 * time.Minute),
			earliestTime: T1.Add(-7 * time.Second),
			cron:         schedule,
			expectErr:    false,
			expectResult: []time.Time{T1},
		},
		{
			// Case 3: known LastScheduleTime, no start needed.
			name:         "Case 3: known LastScheduleTime, no start needed",
			nowTime:      T1.Add(2 * time.Minute),
			earliestTime: T1,
			cron:         schedule,
			expectErr:    false,
			expectResult: []time.Time{},
		},
		{
			// Case 4: known LastScheduleTime, a start needed
			name:         "Case 4: known LastScheduleTime, a start needed",
			nowTime:      T2.Add(5 * time.Minute),
			earliestTime: T1,
			cron:         schedule,
			expectErr:    false,
			expectResult: []time.Time{T2},
		},
		{
			// Case 5: known LastScheduleTime, two starts needed
			name:         "Case 5: known LastScheduleTime, two starts needed",
			nowTime:      T2.Add(5 * time.Minute),
			earliestTime: T1.Add(-1 * time.Hour),
			cron:         schedule,
			expectErr:    false,
			expectResult: []time.Time{T1, T2},
		},
		{
			// Case6: expect err
			name:         "Case6: expect err",
			nowTime:      T2.Add(100 * time.Hour),
			earliestTime: T1.Add(-1 * time.Hour),
			cron:         schedule,
			expectErr:    true,
			expectResult: []time.Time{},
		},
		// add more ..
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := getRecentUnmetScheduleTimes(testCase.earliestTime, testCase.nowTime, testCase.cron, testCase.startingDeadlineSeconds)
			if err != nil && !testCase.expectErr {
				t.Errorf("unexpectErr: %v, testCase: %v", err, testCase)
			}

			if !reflect.DeepEqual(result, testCase.expectResult) {
				t.Errorf("getRecentUnmetScheduleTimes expect: %v, get : %v, testCase: %v", testCase.expectResult, result, testCase)
			}
		})
	}

}

func TestGetExcludeTimes(t *testing.T) {
	// schedule is hourly on the hour
	schedule := "0 * * * ?"
	// T1 is a scheduled start time of that schedule
	T1, err := time.Parse(time.RFC3339, "2016-05-19T10:00:00Z")
	if err != nil {
		t.Errorf("test setup error: %v", err)
	}
	// T2 is a scheduled start time of that schedule after T1
	T2, err := time.Parse(time.RFC3339, "2016-05-19T11:00:00Z")
	if err != nil {
		t.Errorf("test setup error: %v", err)
	}

	testCases := []struct {
		name                    string
		nowTime                 time.Time
		earliestTime            time.Time
		crons                   []string
		startingDeadlineSeconds int32
		expectErr               bool
		expectResult            map[time.Time]int
	}{
		{
			// Case 1: no known start times, and none needed yet.
			name:         "Case 1: no known start times, and none needed yet",
			nowTime:      T1.Add(-7 * time.Minute),
			earliestTime: T1.Add(-10 * time.Minute),
			crons:        []string{schedule},
			expectErr:    false,
			expectResult: nil,
		},
		{
			// Case 2: no known start times, and one needed.
			name:         "Case 2: no known start times, and one needed.",
			nowTime:      T1.Add(2 * time.Minute),
			earliestTime: T1.Add(-7 * time.Second),
			crons:        []string{schedule},
			expectErr:    false,
			expectResult: map[time.Time]int{T1: 1},
		},
		{
			// Case 3: known LastScheduleTime, no start needed.
			name:         "Case 3: known LastScheduleTime, no start needed.",
			nowTime:      T1.Add(2 * time.Minute),
			earliestTime: T1,
			crons:        []string{schedule},
			expectErr:    false,
			expectResult: nil,
		},
		{
			// Case 4: known LastScheduleTime, a start needed
			name:         "Case 4: known LastScheduleTime, a start needed",
			nowTime:      T2.Add(5 * time.Minute),
			earliestTime: T1,
			crons:        []string{schedule},
			expectErr:    false,
			expectResult: map[time.Time]int{T2: 1},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := getExcludeTimes(testCase.earliestTime, testCase.nowTime, testCase.crons, testCase.startingDeadlineSeconds)
			if err != nil && !testCase.expectErr {
				t.Errorf("unexpectErr: %v, testCase: %v", err, testCase)
			}

			if !equalMap(result, testCase.expectResult) {
				t.Errorf("getExcludeTimes expect: %v, get : %v, testCase: %v", testCase.expectResult, result, testCase)
			}
		})
	}

}

func TestShoudFilterTimesWithExcludeTimes(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		excludeTimes map[time.Time]int
		inputTimes   time.Time
		shoudFilter  bool
	}{
		{
			excludeTimes: map[time.Time]int{},
			inputTimes:   now,
			shoudFilter:  false,
		},
		{
			excludeTimes: map[time.Time]int{now: 1},
			inputTimes:   now,
			shoudFilter:  true,
		},
		{
			excludeTimes: map[time.Time]int{now: 1},
			inputTimes:   now.Add(-time.Second * time.Duration(1)),
			shoudFilter:  false,
		},
	}

	for _, testCase := range testCases {
		result := shoudFilterTimesWithExcludeTimes(testCase.excludeTimes, testCase.inputTimes)
		if result != testCase.shoudFilter {
			t.Errorf("shoudFilterTimesWithExcludeTimes err, want: %v, get: %v, testCase: %v", testCase.shoudFilter, result, testCase)
		}
	}
}

func equalMap(map1, map2 map[time.Time]int) bool {
	if len(map1) != len(map2) {
		return false
	}

	for k, v := range map1 {
		if map2[k] != v {
			return false
		}
	}

	return true
}

func TestHaveRunAlready(t *testing.T) {
	testCases := []struct {
		name    string
		cron    cronHPAV1.Cron
		cronHPA *cronHPAV1.CronHPA
		expect  bool
	}{
		{
			name: "case 1 already run",
			cron: cronHPAV1.Cron{Name: "test_normal"},
			cronHPA: &cronHPAV1.CronHPA{
				Status: cronHPAV1.CronHPAStatus{
					Conditions: []cronHPAV1.CronHPAConditions{
						{Name: "test_normal", Status: cronHPAV1.Succeed},
					},
				},
			},
			expect: true,
		},
		{
			name: "case 2 already run with multiple conditons",
			cron: cronHPAV1.Cron{Name: "test_normal"},
			cronHPA: &cronHPAV1.CronHPA{
				Status: cronHPAV1.CronHPAStatus{
					Conditions: []cronHPAV1.CronHPAConditions{
						{Name: "test_normal", Status: cronHPAV1.Succeed},
						{Name: "test_normal2", Status: cronHPAV1.Failed},
						{Name: "test_normal3", Status: cronHPAV1.Submitted},
					},
				},
			},
			expect: true,
		},
		{
			name: "case 3 run with none conditons",
			cron: cronHPAV1.Cron{Name: "test_normal"},
			cronHPA: &cronHPAV1.CronHPA{
				Status: cronHPAV1.CronHPAStatus{
					Conditions: []cronHPAV1.CronHPAConditions{},
				},
			},
			expect: false,
		},
		{
			name: "case 4 not run with multiple conditons",
			cron: cronHPAV1.Cron{Name: "test_normal"},
			cronHPA: &cronHPAV1.CronHPA{
				Status: cronHPAV1.CronHPAStatus{
					Conditions: []cronHPAV1.CronHPAConditions{
						{Name: "test_normal", Status: cronHPAV1.Submitted},
						{Name: "test_normal2", Status: cronHPAV1.Failed},
						{Name: "test_normal3", Status: cronHPAV1.Submitted},
					},
				},
			},
			expect: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := haveRunAlready(testCase.cron, testCase.cronHPA)
			if result != testCase.expect {
				t.Errorf("haveRunAlready err, want: %v, get: %v, case: %v", testCase.expect, result, testCase)
			}
		})
	}
}

func TestUpdateConditions(t *testing.T) {
	testCases := []struct {
		name         string
		newCondition cronHPAV1.CronHPAConditions
		conditions   []cronHPAV1.CronHPAConditions
		expect       []cronHPAV1.CronHPAConditions
	}{
		{
			name:         "case 1 normal update",
			newCondition: cronHPAV1.CronHPAConditions{Name: "test"},
			conditions:   []cronHPAV1.CronHPAConditions{},
			expect: []cronHPAV1.CronHPAConditions{
				{Name: "test"},
			},
		},
		{
			name:         "case 2 normal update",
			newCondition: cronHPAV1.CronHPAConditions{Name: "test"},
			conditions: []cronHPAV1.CronHPAConditions{
				{Name: "test2"},
			},
			expect: []cronHPAV1.CronHPAConditions{
				{Name: "test2"},
				{Name: "test"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := updateConditions(testCase.conditions, testCase.newCondition)
			sort.Slice(result, func(i, j int) bool { return result[i].Name > result[j].Name })
			sort.Slice(testCase.expect, func(i, j int) bool { return testCase.expect[i].Name > testCase.expect[j].Name })
			if !reflect.DeepEqual(result, testCase.expect) {
				t.Errorf("updateConditions err, want: %v, get: %v, case: %v", testCase.expect, result, testCase)
			}
		})
	}
}
