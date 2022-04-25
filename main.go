package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/fatih/color"
)

type agoFn func(time.Time) time.Time

var (
	weekFlag  = flag.Bool("week", false, "aggregate by week")
	monthFlag = flag.Bool("month", false, "aggregate by month")
	yearFlag  = flag.Bool("year", false, "aggregate by year")
	ratioFlag = flag.Bool("ratio", false, "show activities' ratio")
)

func main() {
	flag.Usage = func() {
		usage := `act is a tool to show your GitHub activity.
It aggregates the activities by the specified period and compares them and the previous period.

Example:
$ gh act # Count your activities in the last week
Issue      	 6 (-1)
PullRequest	 5 (0)
Reviews    	10 (-4)

$ gh act -ratio -month # Show your activities' ratio in the last month
Issue      	28% (+2%)
PullRequest	24% (+5%)
Reviews   	48% (-6%)

Usage:`
		fmt.Fprintln(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := act(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func act() error {
	cur, pre, err := aggregate()
	if err != nil {
		return err
	}
	showActivity(cur, pre)
	return nil
}

func aggregate() (Activity, Activity, error) {
	client, err := gh.GQLClient(nil)
	if err != nil {
		return Activity{}, Activity{}, err
	}

	var fn agoFn
	switch {
	case *weekFlag:
		fn = weekAgo
	case *monthFlag:
		fn = monthAgo
	case *yearFlag:
		fn = yearAgo
	default:
		fn = weekAgo
	}
	return getActivities(client, time.Now(), fn)
}

func showActivity(cur, pre Activity) {
	df := countAndDiff
	if *ratioFlag {
		df = ratioAndDiff
	}
	d := df(cur, pre)
	s := fmt.Sprintf("Issue      \t%s\n", d.issue)
	s += fmt.Sprintf("PullRequest\t%s\n", d.pr)
	s += fmt.Sprintf("Reviews    \t%s\n", d.reviews)
	fmt.Print(s)
}

type activityDiff struct {
	issue   string
	pr      string
	reviews string
}

func countAndDiff(cur, pre Activity) activityDiff {
	padFmt := fmt.Sprintf("%%%dv", maxLen(cur.issues, cur.prs, cur.reviews))
	var d activityDiff
	d.issue = fmt.Sprintf(padFmt+" (%s)", cur.issues, countDiff(cur.issues, pre.issues))
	d.pr = fmt.Sprintf(padFmt+" (%s)", cur.prs, countDiff(cur.prs, pre.prs))
	d.reviews = fmt.Sprintf(padFmt+" (%s)", cur.reviews, countDiff(cur.reviews, pre.reviews))
	return d
}

func ratio(nums ...int) []int {
	total := 0.0
	for _, n := range nums {
		total += float64(n)
	}
	if total == 0 {
		return make([]int, len(nums))
	}

	r := make([]int, len(nums))
	for i, n := range nums {
		r[i] = int(math.Round(100.0 * float64(n) / total))
	}
	return r
}

func ratioAndDiff(cur, pre Activity) activityDiff {
	curratio := ratio(cur.issues, cur.prs, cur.reviews)
	preratio := ratio(pre.issues, pre.prs, pre.reviews)

	padFmt := fmt.Sprintf("%%%dv%%%%", maxLen(curratio...))
	var d activityDiff
	d.issue = fmt.Sprintf(padFmt+" (%s)", curratio[0], ratioDiff(curratio[0], preratio[0]))
	d.pr = fmt.Sprintf(padFmt+" (%s)", curratio[1], ratioDiff(curratio[1], preratio[1]))
	d.reviews = fmt.Sprintf(padFmt+" (%s)", curratio[2], ratioDiff(curratio[2], preratio[2]))
	return d
}

func maxLen(nums ...int) int {
	max := 0
	for _, n := range nums {
		if m := len(fmt.Sprint(n)); m > max {
			max = m
		}
	}
	return max
}

func countDiff(n, m int) string {
	return diff(n, m, "")
}

func ratioDiff(n, m int) string {
	return diff(n, m, "%")
}

func diff(n, m int, unit string) string {
	d := n - m
	switch {
	case d == 0:
		return fmt.Sprintf("0%s", unit)
	case d < 0:
		return color.New(color.FgRed).Sprintf("%+d%s", d, unit)
	default:
		return color.New(color.FgGreen).Sprintf("%+d%s", d, unit)
	}
}

type Activity struct {
	issues  int
	prs     int
	reviews int
}

func getActivities(client api.GQLClient, d time.Time, fn agoFn) (Activity, Activity, error) {
	ago := fn(d)
	r1, err := countActivityQuery(client, ago, d)
	if err != nil {
		return Activity{}, Activity{}, err
	}

	r2, err := countActivityQuery(client, fn(ago), ago)
	if err != nil {
		return Activity{}, Activity{}, err
	}

	return r1, r2, nil
}

type ActCountQuery struct {
	OpenIssue struct {
		IssueCount int
	} `graphql:"openIssue: search(query: $queryForOpenIssue, type: ISSUE, first: 1)"`
	OpenPr struct {
		IssueCount int
	} `graphql:"openPr: search(query: $queryForOpenPR, type: ISSUE, first: 1)"`
	ReviewedPr struct {
		IssueCount int
	} `graphql:"reviewedPr: search(query: $queryForReviewedPR, type: ISSUE, first: 1)"`
}

func countActivityQuery(client api.GQLClient, begin, end time.Time) (Activity, error) {
	period := fmt.Sprintf("%s..%s", dateToStr(begin), dateToStr(end))
	var query ActCountQuery
	variables := map[string]interface{}{
		"queryForOpenIssue":  graphql.String(fmt.Sprintf("author:@me is:issue created:%s", period)),
		"queryForOpenPR":     graphql.String(fmt.Sprintf("author:@me is:pr created:%s", period)),
		"queryForReviewedPR": graphql.String(fmt.Sprintf("-author:@me is:pr reviewed-by:@me updated:%s", period)),
	}
	err := client.Query("query", &query, variables)
	if err != nil {
		return Activity{}, err
	}
	act := Activity{
		issues:  query.OpenIssue.IssueCount,
		prs:     query.OpenPr.IssueCount,
		reviews: query.ReviewedPr.IssueCount,
	}
	return act, err
}

func weekAgo(d time.Time) time.Time {
	return d.AddDate(0, 0, -7)
}

func monthAgo(d time.Time) time.Time {
	return d.AddDate(0, -1, 0)
}

func yearAgo(d time.Time) time.Time {
	return d.AddDate(-1, 0, 0)
}

func dateToStr(d time.Time) string {
	return d.Format("2006-01-02")
}
