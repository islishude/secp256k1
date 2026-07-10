# Constant-time base-point window matrix

Date: 2026-07-10  
Go: go1.26.4  
GOOS/GOARCH: darwin/arm64  
CPU: Apple M3 Max

All candidates use fixed loop counts and full table scans. No secret digit is
used as a table index or branch condition.

| Candidate              | Table entries/window | Windows | Projective base multiply | Decision                  |
| ---------------------- | -------------------: | ------: | -----------------------: | ------------------------- |
| unsigned W4            |                   16 |      64 |            about 17.0 us | retained as test oracle   |
| unsigned W5            |                   32 |      52 |            about 18.3 us | rejected                  |
| unsigned W6            |                   64 |      43 |            about 21.7 us | rejected                  |
| signed W5 affine table |                   16 |      52 |            about 14.0 us | improved, then superseded |
| signed W6 affine table |                   32 |      43 |            about 15.0 us | rejected                  |
| signed W5 packed limbs |                   16 |      52 |            about 13.1 us | selected                  |

The selected W5 representation scans generated `[8]uint64` Montgomery coordinate
records directly. The first window initializes a projective point with a masked
infinity selection, so the path performs 51 complete mixed additions rather than 52. The generated word table is also the signed-in source of truth, avoiding a
second loaded copy of the W5 affine table at package initialization.

The final ten-run workload median is 24,632 ns/op for recoverable signing, down
from the 28,619 ns/op baseline, with 0 B/op and 0 allocs/op.
