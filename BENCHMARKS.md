# ⏱️ gram Benchmarks

[← Back to README](./README.md)

> What each benchmark measures, the numbers, and what they say about a tick.

## Environment

- **CPU:** Intel(R) Core(TM) i5-8265U CPU @ 1.60GHz (4 cores, 8 threads)
- **Go version:** 1.27.1
- **OS:** Linux

Every number below is the **median of 5 runs** (`make bench-save`, `-count=5`) at commit `651d29a`,
2026-09-22. This machine drifts by up to ~10% between runs, so a change is judged by running the
old and the new tree **alternately**, never one run each; see [How to benchmark](#how-to-benchmark).

All benchmarks live in [`bench/`](bench/) and build their Stage the way a game does — plugins
installed through a headless `game.Initializer`, entities spawned from kinds — then tick the ECS
directly, with no window. "0 allocs/op" means a steady tick allocates nothing; the small B/op on
the larger scenes is the amortised growth of buffers kept between ticks.

## World tick — `Benchmark_World_*`

`Benchmark_World_Tick` is one tick of the world plugin alone: the decision pass, steering and
velocity folded into each entity's speed, every box moved under the edge rules, and the space
rebuilt from every entity. `Benchmark_World_PositionScan` is the floor under it: reading every
entity's `Base` through a goke query, chunk by chunk.

| Scene | Tick | Position scan |
|:---|---:|---:|
| 1,000 boxes, 20×20 each, on a 4000×4000 torus | 40 µs | 2.4 µs |
| 5,000 boxes, 20×20 each, on the same torus | 199 µs | 12.1 µs |

## Collision tick — `Benchmark_Collision_Tick`

One tick of world plus collision at the collision demo's scales: movement, the space rebuilt,
every overlapping pair found, tested, bounced and pushed apart, with a `CountContacts` behavior
counting. The scene is a 1024×1024 torus covered to a share of its area with square boxes of one
side, each heading somewhere at random, after 120 ticks so the boxes have spread.
"Covering 20%" means the boxes' total area is 20% of the world's.

| Scene | Entities | Tick | Contacts per tick |
|:---|---:|---:|---:|
| boxes 20×20, covering 20% | 524 | 134 µs | 16 |
| boxes 20×20, covering 40% | 1,048 | 534 µs | 91 |
| boxes 10×10, covering 20% | 2,097 | 608 µs | 122 |
| boxes 10×10, covering 40% | 4,194 | 2.32 ms | 695 |
| boxes 8×8, covering 20% | 3,276 | 995 µs | 233 |
| boxes 5×5, covering 20% | 8,388 | 2.88 ms | 916 |

The cost follows the entity count more than the contact count: at the same coverage, halving
the box side quadruples the population and roughly quadruples the tick, while doubling the
coverage at one size quadruples the contacts and the tick alike.

## Sight — `Benchmark_Vision_*`

One tick of the vision plugin alone over a 4000×4000 world: every observer, on a 120-unit
lattice with a 60° cone of radius 200 facing right, scans the shared space and fills its `Seen`.
With outlines, every observer also carries a `SightOutline`, so its view's shape is computed for
drawing.

| Observers | Scan | Scan with outlines |
|---:|---:|---:|
| 100 | 101 µs | 237 µs |
| 500 | 503 µs | 1.27 ms |

About 1 µs per observer to know what it sees, 2.5 µs to also know the shape of its view; both
scale linearly with the observers.

## Key takeaways

* **A tick is the plugins' RunPlan and nothing else.** The engine adds no work of its own per
  entity; what a Stage pays is the sum of the plugins it runs, in the order it runs them.
* **The world tick is the space rebuild plus a walk.** Moving 5,000 entities and handing the space
  every `Base` costs under 200 µs; the query walk under it is 12 µs.
* **Collisions cost by population.** At the demo's default scale (8,388 boxes of 5×5) a tick is
  under 3 ms, well inside the 8.3 ms of a 120 TPS step; the crowd that halves the demo's TPS is the
  40% coverage one, where contacts dominate.
* **Zero allocations once warm.** Every benchmark reports 0 allocs/op after the first ticks have
  grown the buffers.

## How to benchmark

```bash
make bench        # the whole suite once, with allocations
make bench-save   # 5 repeats, raw output under bench_results/ (ignored by git)
```

The benchmarks link Ebitengine but open no window; in CI they run under `xvfb-run` like the
tests. To judge a change on this or any drifting machine, keep a copy of the baseline tree and
run the two alternately, then compare medians:

```bash
cp -r . /tmp/before          # before the change
for i in 1 2; do
  (cd /tmp/before && go test -run xxx -bench . -count 4 ./bench/... | sed 's/^/BEFORE /')
  go test -run xxx -bench . -count 4 ./bench/... | sed 's/^/AFTER  /'
done
```
