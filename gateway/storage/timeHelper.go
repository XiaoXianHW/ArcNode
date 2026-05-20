package storage

import "time"

func timeUnix(ts int64) time.Time { return time.Unix(ts, 0).In(time.Local) }
