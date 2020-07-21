# syng
Will automatically commit and push changes to the repository it is executed in
after the newest change is at least 30s old or at the latest after the oldest
change is 5min old. Useful to keep repositories containing notes up to date
without manually commiting every few seconds/minutes while avoiding to create
commits for every single change to keep overall amount of commits in check.

Time limits can be set via environment variables SYNG_SYNC_AFTER and
SYNG_FORCE_SYNC_AFTER.
