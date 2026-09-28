# Network-Aware Scheduling for Distributed Deep Learning

Course project for Fundamentals of Distributed Computing (University of Tehran, Spring 2026), done with **Taha Majlesi**.

## Problem

Training large models needs **hybrid parallelism** (data, pipeline and tensor parallel) across many GPUs. When several training jobs share one cluster, the scheduler has to decide which GPUs each job gets, how micro-batches are ordered, and when input data is prefetched. It has to do this while the jobs compete for VRAM and for bandwidth on a shared, hierarchical (fat-tree) data-centre network.

## Approach

- **Multi-objective formulation:** lateness, overall makespan, job completion time (JCT), GPU utilisation and peak VRAM.
- **Network model:** transmission and hop-by-hop propagation delays on a converged fat-tree topology.
- **Exact model:** a mixed-integer linear program (MILP) solved with **SCIP**. Constraints cover VRAM, links and bandwidth, GPU assignment, micro-batch ordering, and time bounds, plus pruning constraints to shrink the search space. Inputs are JSON cluster and job configs.
- **Heuristics:** the MILP does not scale to large clusters, so fast heuristics were designed to match the MILP's schedules at a fraction of the runtime.

## Evaluation

The approach was compared against baseline schedulers, with ablations on prefetching strategy, placement awareness, joint vs. decomposed optimisation, data vs. pipeline parallelism, and dispatch rules. Metrics were deadline satisfaction, makespan, fairness, JCT, lateness and peak VRAM.

**Findings**
- Heterogeneous jobs have different traffic and compute patterns and can be overlapped profitably.
- Prefetching only helps when it's done at the right moments.
- Placement awareness noticeably reduces network delays.
- Joint optimisation beats solving placement and scheduling separately.
- Under loose deadlines, data parallelism is preferred over pipeline parallelism.

## Slides

| File | Content |
|------|---------|
| `presentation-1.pptx` | Motivation: why large models need distributed training; the parallelism landscape (Aceso, FlexFlow, Alpa) |
| `presentation-2.pptx` | *On Improving Efficiency of Distributed Deep Learning Training*: I/O-aware data loading (MegaScale-Data), network- and memory-aware placement (NEST) |
| `presentation-3.pptx` | Final presentation: formulation, MILP/SCIP model, heuristics and full evaluation |

> The solver and simulation code are not included in this repository.
