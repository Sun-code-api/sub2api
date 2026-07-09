<template>
  <component :is="isAuthenticated ? AppLayout : 'div'">
    <div
      :class="
        isAuthenticated
          ? ''
          : 'min-h-screen bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950'
      "
    >
      <!-- 未登录时的简易页头 -->
      <header
        v-if="!isAuthenticated"
        class="border-b border-gray-200/50 px-6 py-4 dark:border-dark-800/50"
      >
        <nav class="mx-auto flex max-w-6xl items-center justify-between">
          <router-link to="/home" class="flex items-center gap-3">
            <div class="h-9 w-9 overflow-hidden rounded-xl shadow-md">
              <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
            </div>
            <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
          </router-link>
          <router-link
            to="/login"
            class="inline-flex items-center rounded-full bg-gray-900 px-4 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('modelPlaza.loginForMore') }}
          </router-link>
        </nav>
      </header>

      <div class="mx-auto max-w-6xl px-4 py-6 sm:px-6">
        <!-- 标题（未登录时展示，登录后 AppLayout 自带页头） -->
        <div v-if="!isAuthenticated" class="mb-6 text-center">
          <h1 class="mb-2 text-3xl font-bold text-gray-900 dark:text-white">
            {{ t('modelPlaza.title') }}
          </h1>
          <p class="text-sm text-gray-600 dark:text-dark-400">
            {{ t('modelPlaza.description') }}
          </p>
        </div>

        <!-- 筛选栏 -->
        <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-wrap items-center gap-2">
            <button
              v-for="p in platformTabs"
              :key="p"
              @click="activePlatform = p"
              class="rounded-full px-4 py-1.5 text-sm font-medium transition-colors"
              :class="
                activePlatform === p
                  ? 'bg-primary-500 text-white shadow-sm'
                  : 'bg-white/70 text-gray-600 hover:bg-gray-100 dark:bg-dark-800/70 dark:text-dark-300 dark:hover:bg-dark-700'
              "
            >
              {{ p === 'all' ? t('modelPlaza.allPlatforms') : platformLabel(p) }}
            </button>
          </div>
          <div class="relative w-full sm:w-72">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('modelPlaza.searchPlaceholder')"
              class="input pl-10"
            />
          </div>
        </div>

        <!-- 加载中 -->
        <div v-if="loading" class="flex items-center justify-center py-24">
          <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
        </div>

        <!-- 空状态 -->
        <div
          v-else-if="filteredModels.length === 0"
          class="py-24 text-center text-sm text-gray-500 dark:text-dark-400"
        >
          {{ t('modelPlaza.empty') }}
        </div>

        <!-- 模型卡片 -->
        <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="m in filteredModels"
            :key="m.platform + '/' + m.name"
            class="flex flex-col rounded-2xl border border-gray-200/60 bg-white/70 p-5 backdrop-blur-sm transition-shadow hover:shadow-lg hover:shadow-primary-500/10 dark:border-dark-700/60 dark:bg-dark-800/70"
          >
            <!-- 名称 + 平台 -->
            <div class="mb-3 flex items-start justify-between gap-2">
              <h3 class="break-all text-sm font-semibold text-gray-900 dark:text-white">
                {{ m.name }}
              </h3>
              <span
                class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium"
                :class="platformBadgeClass(m.platform)"
              >
                {{ platformLabel(m.platform) }}
              </span>
            </div>

            <!-- 定价 -->
            <div class="mb-4 space-y-1.5">
              <template v-if="m.pricing">
                <div
                  v-for="row in pricingRows(m.pricing)"
                  :key="row.label"
                  class="flex items-baseline justify-between text-xs"
                >
                  <span class="text-gray-500 dark:text-dark-400">{{ row.label }}</span>
                  <span class="font-mono font-medium text-gray-900 dark:text-white">
                    {{ row.value }}
                    <span class="ml-0.5 font-sans text-[10px] font-normal text-gray-400">{{
                      row.unit
                    }}</span>
                  </span>
                </div>
              </template>
              <div v-else class="text-xs text-gray-400 dark:text-dark-500">
                {{ t('modelPlaza.noPricing') }}
              </div>
            </div>

            <!-- 分组倍率 -->
            <div class="mt-auto border-t border-gray-100 pt-3 dark:border-dark-700/60">
              <div class="mb-1.5 text-[10px] font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">
                {{ t('modelPlaza.groups') }}
              </div>
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="g in m.groups"
                  :key="g.id"
                  class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px]"
                  :class="
                    g.is_exclusive
                      ? 'bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-400 dark:ring-amber-800/50'
                      : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
                  "
                  :title="groupTooltip(g)"
                >
                  {{ g.name }}
                  <span class="font-mono font-semibold" :class="userRateFor(g) !== null ? 'text-primary-600 dark:text-primary-400' : ''">
                    ×{{ formatRate(userRateFor(g) ?? g.rate_multiplier) }}
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 未登录提示 -->
        <div v-if="!isAuthenticated && !loading" class="mt-10 text-center">
          <router-link
            to="/register"
            class="btn btn-primary px-8 py-2.5 text-sm shadow-lg shadow-primary-500/30"
          >
            {{ t('modelPlaza.registerCta') }}
          </router-link>
        </div>
      </div>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import modelPlazaAPI, { type ModelPlazaEntry } from '@/api/modelPlaza'
import userGroupsAPI from '@/api/groups'
import type { UserAvailableGroup, UserSupportedModelPricing } from '@/api/channels'
import { useAuthStore, useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', {
    allowRelative: true,
    allowDataUrl: true
  })
)

const models = ref<ModelPlazaEntry[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const activePlatform = ref('all')

const platformTabs = computed(() => {
  const set = new Set<string>()
  for (const m of models.value) set.add(m.platform)
  return ['all', ...Array.from(set).sort()]
})

const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return models.value.filter((m) => {
    if (activePlatform.value !== 'all' && m.platform !== activePlatform.value) return false
    if (!q) return true
    return (
      m.name.toLowerCase().includes(q) ||
      m.platform.toLowerCase().includes(q) ||
      m.groups.some((g) => g.name.toLowerCase().includes(q))
    )
  })
})

const platformLabels: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok'
}

function platformLabel(p: string): string {
  return platformLabels[p] || p
}

function platformBadgeClass(p: string): string {
  switch (p) {
    case 'anthropic':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'openai':
      return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    case 'gemini':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'antigravity':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-400'
    case 'grok':
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

interface PricingRow {
  label: string
  value: string
  unit: string
}

/** token 单价（$/token）转为 $/MTok 展示。 */
function perMTok(price: number): string {
  const v = price * 1_000_000
  return `$${v >= 100 ? v.toFixed(0) : v.toPrecision(3)}`
}

function pricingRows(p: UserSupportedModelPricing): PricingRow[] {
  const rows: PricingRow[] = []
  const mtok = t('modelPlaza.unitPerMTok')
  if (p.billing_mode === 'per_request' && p.per_request_price !== null) {
    rows.push({
      label: t('modelPlaza.perRequest'),
      value: `$${p.per_request_price}`,
      unit: t('modelPlaza.unitPerRequest')
    })
    return rows
  }
  if (p.input_price !== null) rows.push({ label: t('modelPlaza.input'), value: perMTok(p.input_price), unit: mtok })
  if (p.output_price !== null) rows.push({ label: t('modelPlaza.output'), value: perMTok(p.output_price), unit: mtok })
  if (p.cache_read_price !== null)
    rows.push({ label: t('modelPlaza.cacheRead'), value: perMTok(p.cache_read_price), unit: mtok })
  if (p.cache_write_price !== null)
    rows.push({ label: t('modelPlaza.cacheWrite'), value: perMTok(p.cache_write_price), unit: mtok })
  if (p.image_output_price !== null)
    rows.push({ label: t('modelPlaza.imageOutput'), value: perMTok(p.image_output_price), unit: mtok })
  return rows
}

function userRateFor(g: UserAvailableGroup): number | null {
  const r = userGroupRates.value[g.id]
  return typeof r === 'number' ? r : null
}

function formatRate(r: number): string {
  return Number.isInteger(r) ? r.toFixed(1) : String(r)
}

function groupTooltip(g: UserAvailableGroup): string {
  const parts: string[] = []
  parts.push(g.is_exclusive ? t('modelPlaza.exclusiveGroup') : t('modelPlaza.publicGroup'))
  if (g.peak_rate_enabled) {
    parts.push(
      t('modelPlaza.peakRate', {
        start: g.peak_start,
        end: g.peak_end,
        rate: formatRate(g.peak_rate_multiplier)
      })
    )
  }
  if (userRateFor(g) !== null) parts.push(t('modelPlaza.customRate'))
  return parts.join(' · ')
}

async function load() {
  loading.value = true
  try {
    if (isAuthenticated.value) {
      const [list, rates] = await Promise.all([
        modelPlazaAPI.getModelPlaza(),
        userGroupsAPI.getUserGroupRates().catch(() => ({}) as Record<number, number>)
      ])
      models.value = list
      userGroupRates.value = rates
    } else {
      models.value = await modelPlazaAPI.getPublicModelPlaza()
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  load()
})
</script>
