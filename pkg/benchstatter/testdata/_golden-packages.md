pkg: encoding/gob
goos: darwin
note: hw acceleration enabled

|   name    | old time/op (ns/op) | ±  | new time/op (ns/op) | ±  |  delta  |        ±        |
|-----------|--------------------:|----|--------------------:|----|---------|-----------------|
| GobEncode |            13599100 | 1% |            11789300 | 1% | -13.31% | (p=0.016 n=4+5) |

|   name    | old speed (MB/s) | ±  | new speed (MB/s) | ±  |  delta  |        ±        |
|-----------|-----------------:|----|-----------------:|----|---------|-----------------|
| GobEncode |            56.44 | 1% |           65.108 | 1% | +15.36% | (p=0.016 n=4+5) |

pkg: encoding/json
goos: darwin
note: hw acceleration enabled

|    name    | old time/op (ns/op) | ±  | new time/op (ns/op) | ±  | delta |        ±        |
|------------|--------------------:|----|--------------------:|----|-------|-----------------|
| JSONEncode |            32114300 | 1% |            31761400 | 1% | ~     | (p=0.286 n=4+5) |

|    name    | old speed (MB/s) | ±  | new speed (MB/s) | ±  | delta |        ±        |
|------------|-----------------:|----|-----------------:|----|-------|-----------------|
| JSONEncode |          60.4275 | 1% |           61.102 | 2% | ~     | (p=0.286 n=4+5) |
