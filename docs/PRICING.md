# Pulse Pricing & Frequency Recommendations

Quick-start commercial guidance for selling URL monitoring with webhook/email
notifications.

## Capacity Reality

Pulse currently runs 3 workers at ~200ms/check, roughly **15 checks/second**
per instance. Each monitor with interval `P` seconds consumes `1/P` checks per
second, so:

    monitors per instance = 15 x P

| Min interval | Checks / monitor / month | Monitors per instance | Suggested price / monitor / month |
|---|---|---|---|
| 1s   | 2,592,000 | 15    | Not for sale |
| 5s   | 518,400   | 75    | $3.00-5.00  |
| 10s  | 259,200   | 150   | $2.00-4.00  |
| 30s  | 86,400    | 450   | $1.50-3.00  |
| 60s  | 43,200    | 900   | $0.75-1.50  |
| 5min | 8,640     | 4,500 | $0.10-0.30  |

## The Key Insight

A single account with 10 monitors at a 1-second interval consumes roughly
**2/3 of the instance's total capacity** (10 of 15 checks/second), the same
cost as ~900 monitors at a 1-minute interval. This is why the market standard
(UptimeRobot, Pingdom, Better Stack) never sells sub-minute checks as a
standard option.

## Recommended Minimum Frequency

- **Floor: 60 seconds** (1 check/minute) for paid plans.
- **30 seconds**: premium add-on / higher tier only.
- **1 second**: do not sell externally, not even at enterprise level for now.

## Recommended Pricing Tiers

- **Free**: 3-5 monitors @ 5 min (cheap acquisition hook).
- **Starter ~$9-15/month**: 10-20 monitors @ 1 min (~$1/monitor).
- **Pro ~$29-49/month**: 50-100 monitors @ 1 min + 30s optional add-on.
- **Notifications**: include webhook/email in every tier, capped per day
  (e.g. 100 notifications/day). They are cheap (Resend/SES), not the
  bottleneck.

## Engineering Consequence

Selling capacity means enforcing a minimum frequency. The API must reject
monitors with `IntervalSeconds < 60` (configurable constant) in
`monitor.Service.Create/Update`. Currently there is no validation, so any user
can create a 1-second monitor today.