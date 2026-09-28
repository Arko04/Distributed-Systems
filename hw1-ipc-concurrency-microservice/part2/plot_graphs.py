#!/usr/bin/env python3
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

# Read the CSV file
df = pd.read_csv('part2_results.csv')

# Filter data
cpu = df[df['Workload'] == 'CPU-bound']
mixed = df[df['Workload'] == 'Mixed']

# Set style
plt.style.use('seaborn-v0_8-whitegrid')
plt.rcParams['font.size'] = 12

# ------------------------------------------------------------
# Graph 1: Throughput vs Goroutines (CPU-bound, lines for GOMAXPROCS=1,2,8)
# ------------------------------------------------------------
fig1, ax1 = plt.subplots(figsize=(10, 6))
for procs in [1, 2, 8]:
    data = cpu[cpu['GOMAXPROCS'] == procs]
    ax1.plot(data['Goroutines'], data['ThroughputReqPerSec'], 'o-', label=f'GOMAXPROCS={procs}')
ax1.set_xscale('log')
ax1.set_xlabel('Number of Goroutines (log scale)')
ax1.set_ylabel('Throughput (req/s)')
ax1.set_title('CPU‑bound Workload: Throughput vs Goroutines')
ax1.legend()
ax1.grid(True, which='both', linestyle='--', alpha=0.7)
plt.tight_layout()
plt.savefig('graph1_throughput_cpu.png', dpi=150)
plt.close()
print("Saved graph1_throughput_cpu.png")

# ------------------------------------------------------------
# Graph 2: Total Time vs Goroutines (CPU-bound, GOMAXPROCS=8)
# ------------------------------------------------------------
fig2, ax2 = plt.subplots(figsize=(10, 6))
data = cpu[(cpu['GOMAXPROCS'] == 8)]
ax2.plot(data['Goroutines'], data['TotalTimeMs'], 'o-', color='green')
ax2.set_xlabel('Number of Goroutines')
ax2.set_ylabel('Total Time (ms)')
ax2.set_title('CPU‑bound Workload (GOMAXPROCS=8): Total Execution Time')
ax2.grid(True, linestyle='--', alpha=0.7)
plt.tight_layout()
plt.savefig('graph2_totaltime_cpu_gomax8.png', dpi=150)
plt.close()
print("Saved graph2_totaltime_cpu_gomax8.png")

# ------------------------------------------------------------
# Graph 3: Throughput vs Goroutines (Mixed workload, GOMAXPROCS=8)
# ------------------------------------------------------------
fig3, ax3 = plt.subplots(figsize=(10, 6))
data = mixed[(mixed['GOMAXPROCS'] == 8)]
ax3.plot(data['Goroutines'], data['ThroughputReqPerSec'], 'o-', color='orange')
ax3.set_xlabel('Number of Goroutines')
ax3.set_ylabel('Throughput (req/s)')
ax3.set_title('Mixed Workload (GOMAXPROCS=8): Throughput vs Goroutines')
ax3.grid(True, linestyle='--', alpha=0.7)
plt.tight_layout()
plt.savefig('graph3_throughput_mixed_gomax8.png', dpi=150)
plt.close()
print("Saved graph3_throughput_mixed_gomax8.png")

# ------------------------------------------------------------
# Graph 4: Comparison CPU-bound vs Mixed throughput (GOMAXPROCS=8)
# ------------------------------------------------------------
fig4, ax4 = plt.subplots(figsize=(10, 6))
cpu_data = cpu[cpu['GOMAXPROCS'] == 8]
mixed_data = mixed[mixed['GOMAXPROCS'] == 8]
ax4.plot(cpu_data['Goroutines'], cpu_data['ThroughputReqPerSec'], 'o-', label='CPU‑bound', color='blue')
ax4.plot(mixed_data['Goroutines'], mixed_data['ThroughputReqPerSec'], 's-', label='Mixed', color='red')
ax4.set_xlabel('Number of Goroutines')
ax4.set_ylabel('Throughput (req/s)')
ax4.set_title('Comparison: CPU‑bound vs Mixed Workload (GOMAXPROCS=8)')
ax4.legend()
ax4.grid(True, linestyle='--', alpha=0.7)
plt.tight_layout()
plt.savefig('graph4_comparison.png', dpi=150)
plt.close()
print("Saved graph4_comparison.png")

print("\nAll graphs saved. You can now include the PNG files in your report.pdf")