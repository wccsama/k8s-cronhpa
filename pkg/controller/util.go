package controller

import (
	"fmt"
	"time"

	cronutil "github.com/robfig/cron"
	"k8s.io/klog"

	cronHPAV1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
)

// just for unit test
type timeNow interface {
	now() time.Time
}

type timePackage struct{}

func (tp *timePackage) now() time.Time {
	return time.Now()
}

// getRecentUnmetScheduleTimes gets a slice of times (from oldest to latest) that have passed when a Job should have started but did not.
//
// If there are too many (>100) unstarted times, just give up and return an empty slice.
// If there were missed times prior to the last known start time, then those are not returned.
func getRecentUnmetScheduleTimes(earliestTime, now time.Time, cron string, startingDeadlineSeconds int32) ([]time.Time, error) {
	starts := []time.Time{}
	t := time.Time{}
	sched, err := cronutil.ParseStandard(cron)
	if err != nil {
		return starts, fmt.Errorf("unparseable schedule: %v : %v", cron, err)
	}

	if startingDeadlineSeconds != 0 {
		// Controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(startingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}

	if earliestTime.After(now) {
		return []time.Time{}, nil
	}

	for t = sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		starts = append(starts, t)
		// An object might miss several starts. For example, if
		// controller gets wedged on friday at 5:01pm when everyone has
		// gone home, and someone comes in on tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		//
		// I've somewhat arbitrarily picked 100, as more than 80,
		// but less than "lots".

		// TODO: 100 may not be enough for the controller
		if len(starts) > 100 {
			// We can't get the most recent times so just return an empty slice
			return []time.Time{}, fmt.Errorf("too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew")
		}
	}

	return starts, nil
}

func getExcludeTimes(earliestTime, now time.Time, crons []string, startingDeadlineSeconds int32) (map[time.Time]int, error) {
	result := make(map[time.Time]int, 0)
	// TODO: multiple crons mean 'and' or 'or' ?
	for _, cron := range crons {
		times, err := getRecentUnmetScheduleTimes(earliestTime, now, cron, startingDeadlineSeconds)
		if err != nil {
			klog.Errorf("getRecentUnmetScheduleTimes err in getExcludeTimes, err: %v", err)
			return nil, err
		}

		if len(times) == 0 {
			continue
		}

		if _, ok := result[times[len(times)-1]]; !ok {
			result[times[len(times)-1]] = 1
		}
	}

	return result, nil
}

func shoudFilterTimesWithExcludeTimes(excludeTimes map[time.Time]int, timeBatch time.Time) bool {
	if len(excludeTimes) == 0 {
		return false
	}

	if _, ok := excludeTimes[timeBatch]; ok {
		return true
	}

	return false
}

func getLatestScheduledTime(cronhpa *cronHPAV1.CronHPA) time.Time {
	if cronhpa.Status.LastScheduleTime != nil {
		return cronhpa.Status.LastScheduleTime.Time
	}

	return cronhpa.CreationTimestamp.Time
}

func getCronHPAFullName(cronhpa *cronHPAV1.CronHPA) string {
	return cronhpa.Namespace + "/" + cronhpa.Name
}

func convertConditionsToMap(conditions []cronHPAV1.CronHPAConditions) map[string]cronHPAV1.CronHPAConditions {
	conditionMap := make(map[string]cronHPAV1.CronHPAConditions)
	for _, condition := range conditions {
		conditionMap[condition.Name] = condition
	}
	return conditionMap
}

// haveRunAlready returns the status whether the job has been run.
func haveRunAlready(cron cronHPAV1.Cron, cronHPA *cronHPAV1.CronHPA) bool {
	conditionMap := convertConditionsToMap(cronHPA.Status.Conditions)
	if condition, ok := conditionMap[cron.Name]; ok {
		// only succeed is meaning to run once
		if condition.Status == cronHPAV1.Succeed {
			return true
		}
	}

	return false
}

func updateConditions(conditions []cronHPAV1.CronHPAConditions, newCondition cronHPAV1.CronHPAConditions) []cronHPAV1.CronHPAConditions {
	result := make([]cronHPAV1.CronHPAConditions, 0)

	conditionMap := convertConditionsToMap(conditions)
	conditionMap[newCondition.Name] = newCondition
	for _, condition := range conditionMap {
		result = append(result, condition)
	}
	return result
}
