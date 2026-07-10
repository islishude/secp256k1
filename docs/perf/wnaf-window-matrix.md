# wNAF window matrix

Date: 2026-07-10  
Go: go1.26.4  
GOOS/GOARCH: darwin/arm64  
CPU: Apple M3 Max

Each entry is the median of five runs. The first matrix fixes the generator
window at 13.

| Variable window | Hot verify | Cold compressed verify | Double scalar | Public-key odd table |
| --------------: | ---------: | ---------------------: | ------------: | -------------------: |
|               5 |  35,572 ns |              48,889 ns |     32,104 ns |             7,646 ns |
|               6 |  34,500 ns |              50,921 ns |     31,746 ns |            10,723 ns |
|               7 |  32,465 ns |              55,108 ns |     30,439 ns |            16,713 ns |
|               8 |  32,401 ns |              67,356 ns |     29,732 ns |            28,907 ns |

The second matrix fixes the variable window at 8.

| Generator window | Hot verify | Double scalar | Generator entries per table |
| ---------------: | ---------: | ------------: | --------------------------: |
|               11 |  32,617 ns |     30,948 ns |                         512 |
|               12 |  32,689 ns |     29,950 ns |                       1,024 |
|               13 |  32,401 ns |     29,732 ns |                       2,048 |
|               14 |  31,574 ns |     29,329 ns |                       4,096 |

After the mixed-add and doubling changes, the final code was retested at variable
windows 7 and 8. Window 8 measured about 30.0 microseconds for hot verification
versus about 30.9 microseconds at window 7, a stable improvement of approximately
3%. Compressed cold verification at window 8 was about 63.2 microseconds, still
better than the 70.7-microsecond baseline but slower than window 7. Because the
execution plan prioritizes verification throughput, the final defaults are
variable window 8 and generator window 13. Generator window 14 remains rejected:
its gain was below 3% while doubling both generator tables.

The current array-of-structures table layout was retained. Each lookup consumes
both coordinates together, so splitting x and y would add independent address
streams without eliminating coordinate loads. No production layout change was
justified without a measured improvement.
