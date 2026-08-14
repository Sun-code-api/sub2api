<template>
  <div v-if="visible" class="mt-1 space-y-1">
    <div class="flex flex-wrap items-center gap-1.5">
      <button
        type="button"
        class="inline-flex items-center gap-0.5 rounded px-1.5 py-0.5 text-[10px] font-medium text-cyan-700 transition-colors hover:bg-cyan-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-cyan-300 dark:hover:bg-cyan-900/30"
        :disabled="loading"
        :title="t('admin.accounts.usageWindow.opencodeProbeTooltip')"
        @click="handleQuery"
      >
        <svg
          class="h-2.5 w-2.5"
          :class="{ 'animate-spin': loading }"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
        {{ t('admin.accounts.usageWindow.opencodeProbe') }}
      </button>
      <span
        v-for="w in windows"
        :key="w.key"
        :class="badgeClass(w.percent, w.status)"
        :title="w.title"
        class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium"
      >
        {{ w.label }} {{ formatPercent(w.percent) }}
      </span>
    </div>
    <div
      v-if="refreshedHint"
      class="text-[10px] text-gray-500 dark:text-gray-400"
    >
      {{ refreshedHint }}
    </div>
    <div v-if="error" class="truncate text-[10px] text-red-600 dark:text-red-400" :title="error">
      {{ truncatedError }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()

const emit = defineEmits<{
  'account-updated': [account: Account]
}>()

const { t } = useI18n()

const extra = computed(() => (props.account.extra ?? {}) as Record<string, unknown>)
const visible = computed(() => props.account.platform === 'opencode')
const loading = ref(false)
const error = ref<string | null>(null)

function num(v: unknown): number {
  if (typeof v === 'number') return v
  if (typeof v === 'string') {
    const n = parseFloat(v)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

const hasSnapshot = computed(() => str(extra.value.opencode_quota_refreshed_at).length > 0)

const windows = computed(() => {
  if (!hasSnapshot.value) return []
  const e = extra.value
  return [
    windowView('5h', '5H', num(e.opencode_quota_5h_pct), str(e.opencode_quota_5h_status), str(e.opencode_quota_5h_reset_at)),
    windowView('w', 'W', num(e.opencode_quota_weekly_pct), str(e.opencode_quota_weekly_status), str(e.opencode_quota_weekly_reset_at)),
    windowView('m', 'M', num(e.opencode_quota_monthly_pct), str(e.opencode_quota_monthly_status), str(e.opencode_quota_monthly_reset_at))
  ]
})

function windowView(key: string, label: string, percent: number, status: string, resetAt: string) {
  const parts = [`${label}: ${formatPercent(percent)}`]
  if (status) parts.push(status)
  if (resetAt) parts.push(`reset ${resetAt}`)
  return { key, label, percent, status, title: parts.join(' · ') }
}

const refreshedHint = computed(() => {
  const at = str(extra.value.opencode_quota_refreshed_at)
  if (!at) return ''
  const plan = str(extra.value.opencode_plan)
  return plan ? `${plan} · ${at}` : at
})

const truncatedError = computed(() => {
  if (!error.value) return ''
  return error.value.length > 80 ? `${error.value.slice(0, 80)}...` : error.value
})

function formatPercent(v: number): string {
  return `${Math.round(v)}%`
}

function badgeClass(percent: number, status: string): string {
  if (status === 'rate-limited' || percent >= 90) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  }
  if (percent >= 60) {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
}

const extractErrorMessage = (e: unknown): string => {
  const err = e as {
    message?: string
    reason?: string
    response?: { data?: { message?: string; error?: string } }
  }
  return (
    err?.message ||
    err?.reason ||
    err?.response?.data?.message ||
    err?.response?.data?.error ||
    t('common.error')
  )
}

const handleQuery = async () => {
  if (loading.value) return
  loading.value = true
  error.value = null
  try {
    const result = await adminAPI.accounts.queryOpencodeQuota(props.account.id)
    emit('account-updated', {
      ...props.account,
      extra: {
        ...(props.account.extra ?? {}),
        ...result.extra
      }
    })
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.account.id,
  () => {
    error.value = null
    loading.value = false
  }
)
</script>
