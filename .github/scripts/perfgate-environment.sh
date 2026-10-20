#!/usr/bin/env bash
# Optional diagnostics only: report failed reads without changing the gate's
# exit status or any runtime setting. The caller owns output-file errors.
set -u

report() {
  local label="$1"
  shift
  printf '\n--- %s ---\n' "$label"
  if "$@" 2>&1; then
    printf '[exit=0]\n'
  else
    printf '[unavailable exit=%s]\n' "$?"
  fi
}

printf '\n=== phase=%s utc=%s ===\n' "${1:-unknown}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'run_id=%s attempt=%s runner=%s image=%s image_version=%s\n' \
  "${GITHUB_RUN_ID:-unset}" "${GITHUB_RUN_ATTEMPT:-unset}" "${RUNNER_NAME:-unset}" \
  "${ImageOS:-unset}" "${ImageVersion:-unset}"
printf 'GOMAXPROCS=%s GOGC=%s GOMEMLIMIT=%s\n' \
  "${GOMAXPROCS:-unset}" "${GOGC:-unset}" "${GOMEMLIMIT:-unset}"
report 'checkout SHA' git rev-parse HEAD
report 'Go version' go version
report 'CPU model and topology' lscpu
report 'available CPUs' nproc
report 'clock ticks per second' getconf CLK_TCK
report 'load' cat /proc/loadavg
report 'CPU counters (cumulative)' head -n 1 /proc/stat
report 'CPU pressure' cat /proc/pressure/cpu
report 'available memory' awk '/^(MemAvailable|MemTotal):/' /proc/meminfo
report 'current cgroup membership' cat /proc/self/cgroup
report 'cgroup mounts' awk '$0 ~ / - cgroup/ {print}' /proc/self/mountinfo

# Ubuntu hosted runners use the unified hierarchy at /sys/fs/cgroup. Record
# the current group and its ancestors because a parent can impose a tighter
# quota. If this layout differs, retain the membership/mount data and mark it.
group_path=$(awk -F: '$1 == "0" && $2 == "" {print $3}' /proc/self/cgroup 2>/dev/null)
group_dir="/sys/fs/cgroup${group_path}"
if [[ -n "$group_path" && -d "$group_dir" ]]; then
  group_dir="${group_dir%/}"
  while [[ "$group_dir" == /sys/fs/cgroup* ]]; do
    for counter in cpu.max cpu.stat cpu.pressure cpuset.cpus.effective; do
      report "cgroup $group_dir/$counter (cumulative where applicable)" cat "$group_dir/$counter"
    done
    [[ "$group_dir" == /sys/fs/cgroup ]] && break
    group_dir="${group_dir%/*}"
  done
else
  printf '\n[unavailable: current unified cgroup at /sys/fs/cgroup; inspect membership and mounts above]\n'
fi

# Attribute server counters to the compiled gateway, not its go-run parent.
gateway_pids=$(pgrep -x gateway 2>/dev/null)
if [[ -n "$gateway_pids" ]]; then
  for pid in $gateway_pids; do
    case "$pid" in
      *[!0-9]*) continue ;;
    esac
    report "gateway pid=$pid CPU affinity" taskset -pc "$pid"
    report "gateway pid=$pid cgroup membership" cat "/proc/$pid/cgroup"
    report "gateway pid=$pid counters (cumulative)" cat "/proc/$pid/stat"
    report "gateway pid=$pid scheduler counters (cumulative)" cat "/proc/$pid/schedstat"
    report "gateway pid=$pid status" awk '/^(Name|Pid|PPid|VmRSS|Threads|voluntary_ctxt_switches|nonvoluntary_ctxt_switches):/' "/proc/$pid/status"
  done
else
  printf '\n[unavailable: compiled gateway process not found]\n'
fi
