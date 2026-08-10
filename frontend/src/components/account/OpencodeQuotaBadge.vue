<template>
  <div v-if="visible" class="mt-1 flex flex-wrap items-center gap-1.5">
    <span
      v-for="w in windows"
      :key="w.key"
      :class="badgeClass(w.percent)"
      :title="w.title"
      class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium"
    >
      {{ w.label }} {{ formatPercent(w.percent) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()

const extra = computed(() => (props.account.extra ?? {}) as Record<string, unknown>)

const visible = computed(() => props.account.platform === 'opencode')

function num(v: unknown): number {
  if (typeof v === 'number') return v
  if (typeof v === 'string') {
    const n = parseFloat(v)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}

const windows = computed(() => {
  const e = extra.value
  return [
    {
      key: '5h',
      label: '5H',
      percent: num(e.opencode_quota_5h_pct),
      title: `5H window: ${formatPercent(num(e.opencode_quota_5h_pct))} used`
    },
    {
      key: 'w',
      label: 'W',
      percent: num(e.opencode_quota_weekly_pct),
      title: `Weekly window: ${formatPercent(num(e.opencode_quota_weekly_pct))} used`
    },
    {
      key: 'm',
      label: 'M',
      percent: num(e.opencode_quota_monthly_pct),
      title: `Monthly window: ${formatPercent(num(e.opencode_quota_monthly_pct))} used`
    }
  ]
})

function formatPercent(v: number): string {
  return `${Math.round(v)}%`
}

function badgeClass(percent: number): string {
  if (percent >= 90) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  }
  if (percent >= 60) {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
}
</script>
