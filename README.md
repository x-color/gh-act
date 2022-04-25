# gh-act

**act** is a tool to show your GitHub activity.
It aggregates specified period and compare them and the previous period.

## Usage

Count your activities in the last week.

```console
$ gh act
Issue      	 6 (-1)
PullRequest	 5 (0)
Reviews    	10 (+4)
```

This means
- You've created 6 issues and 1 less than the last period.
- You've created 5 pull requests as same number as the last period.
- You've reviewed 10 pull requests 4 more than the last period.


Run `gh act -help` to show more details.

```console
act is a tool to show your GitHub activity.
It aggregates specified period and compare them and the previous period compare them and the previous period.

Example:
$ gh act # Count your activities in the last week
Issue      	 6 (-1)
PullRequest	 5 (0)
Reviews    	10 (-4)

$ gh act -ratio # Show your activities' ratio
Issue      	28% (+2%)
PullRequest	24% (+5%)
Reviews   	48% (-6%)

Usage:
  -month
    	aggregate by month
  -ratio
    	show activities' ratio
  -week
    	aggregate by week
  -year
    	aggregate by year
```

## Installation

```console
$ gh extension install x-color/gh-act
```

## LICENSE

MIT
