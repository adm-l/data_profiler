# Data model

`jobs` tracks asynchronous work.

`profiles` stores immutable JSON snapshots per completed job.

A future production schema should normalize datasets, sources, schedules, rules, alerts, and profile snapshots so multiple runs for the same dataset can be compared without relying on job IDs.
