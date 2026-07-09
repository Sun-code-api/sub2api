<template>
  <AppLayout>
    <div class="space-y-5">
      <!-- Apple 风格 hero 横幅：柔和极光渐变 + 毛玻璃图标 + 大标题 -->
      <div class="plaza-hero relative overflow-hidden rounded-3xl px-8 py-10">
        <div class="plaza-hero-orb plaza-hero-orb-a" aria-hidden="true"></div>
        <div class="plaza-hero-orb plaza-hero-orb-b" aria-hidden="true"></div>
        <div class="relative flex items-center gap-5">
          <div
            class="plaza-hero-icon flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl"
          >
            <Icon name="sparkles" size="lg" class="text-white" />
          </div>
          <div class="min-w-0">
            <h1 class="plaza-hero-title text-3xl font-semibold text-gray-900 dark:text-white">
              {{ t('modelPlaza.title') }}
            </h1>
            <p class="mt-1.5 text-[15px] leading-relaxed text-gray-500 dark:text-dark-300">
              {{ t('modelPlaza.bannerDescription') }}
            </p>
          </div>
          <div class="ml-auto hidden shrink-0 sm:block">
            <span class="plaza-count-pill">
              {{ t('modelPlaza.modelsCount', { count: filteredModels.length }) }}
            </span>
          </div>
        </div>
      </div>

      <!-- 毛玻璃工具栏 -->
      <div class="plaza-toolbar sticky top-2 z-10 rounded-2xl p-3">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div class="min-w-0 flex-1">
            <SearchInput v-model="searchQuery" :placeholder="t('modelPlaza.searchPlaceholder')" />
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="w-44">
              <Select
                v-model="selectedGroupId"
                :options="groupOptions"
                :placeholder="t('modelPlaza.allGroups')"
              />
            </div>
            <div class="w-36">
              <Select
                v-model="selectedProvider"
                :options="providerOptions"
                :placeholder="t('modelPlaza.provider')"
              />
            </div>
            <div class="w-32">
              <Select
                v-model="selectedType"
                :options="typeOptions"
                :placeholder="t('modelPlaza.type')"
              />
            </div>
            <div class="w-36">
              <Select v-model="sortBy" :options="sortOptions" />
            </div>
            <!-- iOS 分段控件 -->
            <div class="plaza-segment relative flex items-center rounded-xl p-0.5">
              <span
                class="plaza-segment-thumb"
                :class="viewMode === 'list' ? 'translate-x-full' : 'translate-x-0'"
                aria-hidden="true"
              ></span>
              <button
                class="plaza-segment-btn"
                :class="viewMode === 'card' ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-dark-400'"
                :title="t('modelPlaza.cardView')"
                @click="viewMode = 'card'"
              >
                <Icon name="grid" size="sm" />
              </button>
              <button
                class="plaza-segment-btn"
                :class="viewMode === 'list' ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-dark-400'"
                :title="t('modelPlaza.listView')"
                @click="viewMode = 'list'"
              >
                <Icon name="menu" size="sm" />
              </button>
            </div>
          </div>
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

      <!-- 卡片视图 -->
      <div v-else-if="viewMode === 'card'" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="(m, i) in filteredModels"
          :key="m.platform + '/' + m.name"
          class="plaza-card flex flex-col rounded-2xl p-5"
          :style="{ '--enter-delay': `${Math.min(i, 11) * 25}ms` }"
        >
          <!-- 提供商 + 状态 -->
          <div class="mb-1.5 flex items-center justify-between">
            <span
              class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500"
              >{{ m.platform }}</span
            >
            <span
              class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-[11px] font-medium text-emerald-600 dark:text-emerald-400"
            >
              <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
              {{ t('modelPlaza.available') }}
            </span>
          </div>
          <h3
            class="mb-4 break-all text-[15px] font-semibold tracking-tight text-gray-900 dark:text-white"
          >
            {{ m.name }}
          </h3>

          <!-- 分色价格块 -->
          <div v-if="m.pricing" class="mb-4 grid grid-cols-2 gap-2">
            <div
              v-for="block in priceBlocks(m.pricing)"
              :key="block.label"
              class="rounded-xl px-3.5 py-2.5"
              :class="block.class"
            >
              <div class="text-[10px] font-medium" :class="block.labelClass">{{ block.label }}</div>
              <div
                class="mt-0.5 text-[15px] font-semibold tabular-nums tracking-tight text-gray-900 dark:text-white"
              >
                {{ block.value }}
              </div>
              <div class="text-[10px] text-gray-400 dark:text-dark-500">{{ block.unit }}</div>
            </div>
          </div>
          <div v-else class="mb-4 text-xs text-gray-400 dark:text-dark-500">
            {{ t('modelPlaza.noPricing') }}
          </div>

          <!-- 底部 chips：类型 + 分组倍率 -->
          <div class="mt-auto flex flex-wrap items-center gap-1.5 border-t border-gray-100 pt-3 dark:border-dark-700/60">
            <span
              class="rounded bg-sky-50 px-1.5 py-0.5 text-[10px] font-medium text-sky-600 dark:bg-sky-900/30 dark:text-sky-400"
            >
              {{ typeLabel(m) }}
            </span>
            <span
              v-for="g in visibleGroups(m)"
              :key="g.id"
              class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px]"
              :class="
                g.is_exclusive
                  ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
                  : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
              "
              :title="groupTooltip(g)"
            >
              {{ g.name }}
              <span
                class="font-mono font-semibold"
                :class="userRateFor(g) !== null ? 'text-primary-600 dark:text-primary-400' : ''"
                >{{ formatRate(userRateFor(g) ?? g.rate_multiplier) }}x</span
              >
            </span>
            <button
              v-if="m.groups.length > groupChipLimit"
              class="rounded px-1.5 py-0.5 text-[10px] font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
              @click="toggleExpand(m)"
            >
              {{
                isExpanded(m)
                  ? t('modelPlaza.collapse')
                  : `+${m.groups.length - groupChipLimit}`
              }}
            </button>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div
        v-else
        class="plaza-card overflow-x-auto rounded-2xl"
      >
        <table class="w-full min-w-[760px] text-left text-sm">
          <thead>
            <tr class="border-b border-gray-100 text-xs text-gray-400 dark:border-dark-700 dark:text-dark-500">
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.model') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.provider') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.input') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.output') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.cacheRead') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.cacheWrite') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('modelPlaza.groups') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="m in filteredModels"
              :key="m.platform + '/' + m.name"
              class="border-b border-gray-50 last:border-0 dark:border-dark-700/50"
            >
              <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ m.name }}</td>
              <td class="px-4 py-3 text-gray-500 dark:text-dark-400">{{ m.platform }}</td>
              <td class="px-4 py-3 tabular-nums text-gray-900 dark:text-white">
                {{ m.pricing?.input_price != null ? perMTok(m.pricing.input_price) : '—' }}
              </td>
              <td class="px-4 py-3 tabular-nums text-gray-900 dark:text-white">
                {{ m.pricing?.output_price != null ? perMTok(m.pricing.output_price) : '—' }}
              </td>
              <td class="px-4 py-3 tabular-nums text-sky-600 dark:text-sky-400">
                {{ m.pricing?.cache_read_price != null ? perMTok(m.pricing.cache_read_price) : '—' }}
              </td>
              <td class="px-4 py-3 tabular-nums text-pink-600 dark:text-pink-400">
                {{ m.pricing?.cache_write_price != null ? perMTok(m.pricing.cache_write_price) : '—' }}
              </td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="g in m.groups"
                    :key="g.id"
                    class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px]"
                    :class="
                      g.is_exclusive
                        ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
                        : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
                    "
                    :title="groupTooltip(g)"
                  >
                    {{ g.name }}
                    <span class="font-mono font-semibold"
                      >{{ formatRate(userRateFor(g) ?? g.rate_multiplier) }}x</span
                    >
                  </span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import modelPlazaAPI, { type ModelPlazaEntry } from '@/api/modelPlaza'
import userGroupsAPI from '@/api/groups'
import type { UserAvailableGroup, UserSupportedModelPricing } from '@/api/channels'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<ModelPlazaEntry[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)

const searchQuery = ref('')
const selectedGroupId = ref<string>('all')
const selectedProvider = ref<string>('all')
const selectedType = ref<string>('all')
const sortBy = ref<string>('name')
const viewMode = ref<'card' | 'list'>('card')

const groupChipLimit = 3
const expandedKeys = ref<Set<string>>(new Set())

function modelKey(m: ModelPlazaEntry): string {
  return m.platform + '/' + m.name
}
function isExpanded(m: ModelPlazaEntry): boolean {
  return expandedKeys.value.has(modelKey(m))
}
function toggleExpand(m: ModelPlazaEntry) {
  const key = modelKey(m)
  const next = new Set(expandedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedKeys.value = next
}
function visibleGroups(m: ModelPlazaEntry): UserAvailableGroup[] {
  return isExpanded(m) ? m.groups : m.groups.slice(0, groupChipLimit)
}

const groupOptions = computed(() => {
  const byId = new Map<number, UserAvailableGroup>()
  for (const m of models.value) {
    for (const g of m.groups) if (!byId.has(g.id)) byId.set(g.id, g)
  }
  const groups = Array.from(byId.values()).sort((a, b) => a.name.localeCompare(b.name))
  return [
    { value: 'all', label: t('modelPlaza.allGroups') },
    ...groups.map((g) => ({
      value: String(g.id),
      label: `${g.name} · ${formatRate(userRateFor(g) ?? g.rate_multiplier)}x · ${g.platform}`
    }))
  ]
})

const providerOptions = computed(() => {
  const set = new Set<string>()
  for (const m of models.value) set.add(m.platform)
  return [
    { value: 'all', label: t('modelPlaza.allProviders') },
    ...Array.from(set)
      .sort()
      .map((p) => ({ value: p, label: p }))
  ]
})

const typeOptions = computed(() => [
  { value: 'all', label: t('modelPlaza.allTypes') },
  { value: 'chat', label: t('modelPlaza.typeChat') },
  { value: 'per_request', label: t('modelPlaza.typePerRequest') }
])

const sortOptions = computed(() => [
  { value: 'name', label: t('modelPlaza.sortName') },
  { value: 'input_asc', label: t('modelPlaza.sortInputAsc') },
  { value: 'input_desc', label: t('modelPlaza.sortInputDesc') }
])

function modelType(m: ModelPlazaEntry): 'chat' | 'per_request' {
  return m.pricing?.billing_mode === 'per_request' ? 'per_request' : 'chat'
}

function typeLabel(m: ModelPlazaEntry): string {
  return modelType(m) === 'per_request' ? t('modelPlaza.typePerRequest') : t('modelPlaza.typeChat')
}

const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const groupId = selectedGroupId.value
  const list = models.value.filter((m) => {
    if (selectedProvider.value !== 'all' && m.platform !== selectedProvider.value) return false
    if (selectedType.value !== 'all' && modelType(m) !== selectedType.value) return false
    if (groupId !== 'all' && !m.groups.some((g) => String(g.id) === groupId)) return false
    if (!q) return true
    return (
      m.name.toLowerCase().includes(q) ||
      m.platform.toLowerCase().includes(q) ||
      m.groups.some((g) => g.name.toLowerCase().includes(q))
    )
  })
  const sorted = [...list]
  if (sortBy.value === 'name') {
    sorted.sort((a, b) => a.name.localeCompare(b.name))
  } else {
    const dir = sortBy.value === 'input_asc' ? 1 : -1
    sorted.sort((a, b) => {
      const pa = a.pricing?.input_price ?? Number.POSITIVE_INFINITY
      const pb = b.pricing?.input_price ?? Number.POSITIVE_INFINITY
      return (pa - pb) * dir
    })
  }
  return sorted
})

interface PriceBlock {
  label: string
  value: string
  unit: string
  class: string
  labelClass: string
}

/** token 单价（$/token）转为 $/M tokens 展示。 */
function perMTok(price: number): string {
  const v = price * 1_000_000
  return `$${v >= 100 ? v.toFixed(0) : v.toPrecision(3)}`
}

function priceBlocks(p: UserSupportedModelPricing): PriceBlock[] {
  const blocks: PriceBlock[] = []
  const mtok = t('modelPlaza.unitPerMTok')
  const plain = 'bg-gray-50 dark:bg-dark-700/50'
  const plainLabel = 'text-gray-400 dark:text-dark-500'
  if (p.billing_mode === 'per_request' && p.per_request_price !== null) {
    blocks.push({
      label: t('modelPlaza.perRequest'),
      value: `$${p.per_request_price}`,
      unit: t('modelPlaza.unitPerRequest'),
      class: plain,
      labelClass: plainLabel
    })
    return blocks
  }
  if (p.input_price !== null)
    blocks.push({ label: t('modelPlaza.input'), value: perMTok(p.input_price), unit: mtok, class: plain, labelClass: plainLabel })
  if (p.output_price !== null)
    blocks.push({ label: t('modelPlaza.output'), value: perMTok(p.output_price), unit: mtok, class: plain, labelClass: plainLabel })
  if (p.cache_read_price !== null)
    blocks.push({
      label: t('modelPlaza.cacheRead'),
      value: perMTok(p.cache_read_price),
      unit: mtok,
      class: 'bg-sky-50 dark:bg-sky-900/20',
      labelClass: 'text-sky-500 dark:text-sky-400'
    })
  if (p.cache_write_price !== null)
    blocks.push({
      label: t('modelPlaza.cacheWrite'),
      value: perMTok(p.cache_write_price),
      unit: mtok,
      class: 'bg-pink-50 dark:bg-pink-900/20',
      labelClass: 'text-pink-500 dark:text-pink-400'
    })
  if (p.image_output_price !== null)
    blocks.push({
      label: t('modelPlaza.imageOutput'),
      value: perMTok(p.image_output_price),
      unit: mtok,
      class: 'bg-violet-50 dark:bg-violet-900/20',
      labelClass: 'text-violet-500 dark:text-violet-400'
    })
  return blocks
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
    const [list, rates] = await Promise.all([
      modelPlazaAPI.getModelPlaza(),
      userGroupsAPI.getUserGroupRates().catch(() => ({}) as Record<number, number>)
    ])
    models.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.plaza-hero {
  background:
    radial-gradient(120% 180% at 0% 0%, rgba(99, 102, 241, 0.12), transparent 55%),
    radial-gradient(120% 180% at 100% 0%, rgba(236, 72, 153, 0.1), transparent 55%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(255, 255, 255, 0.6));
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
:global(.dark .plaza-hero) {
  background:
    radial-gradient(120% 180% at 0% 0%, rgba(99, 102, 241, 0.22), transparent 55%),
    radial-gradient(120% 180% at 100% 0%, rgba(236, 72, 153, 0.16), transparent 55%),
    linear-gradient(180deg, rgba(30, 33, 44, 0.9), rgba(30, 33, 44, 0.6));
  border-color: rgba(255, 255, 255, 0.08);
}
.plaza-hero-orb {
  position: absolute;
  border-radius: 9999px;
  filter: blur(60px);
  opacity: 0.5;
  pointer-events: none;
}
.plaza-hero-orb-a {
  width: 260px;
  height: 260px;
  left: -60px;
  top: -120px;
  background: radial-gradient(circle, rgba(129, 140, 248, 0.5), transparent 70%);
}
.plaza-hero-orb-b {
  width: 300px;
  height: 300px;
  right: -80px;
  bottom: -160px;
  background: radial-gradient(circle, rgba(244, 114, 182, 0.4), transparent 70%);
}
.plaza-hero-icon {
  background: linear-gradient(145deg, #6366f1, #a855f7);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.35),
    0 8px 24px rgba(139, 92, 246, 0.35);
}
.plaza-hero-title {
  letter-spacing: -0.022em;
  line-height: 1.1;
}
.plaza-count-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.375rem 0.875rem;
  border-radius: 9999px;
  font-size: 13px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  color: rgb(75 85 99);
  background: rgba(255, 255, 255, 0.65);
  backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06);
}
:global(.dark .plaza-count-pill) {
  color: rgb(209 213 219);
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.1);
}
.plaza-toolbar {
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.05);
}
:global(.dark .plaza-toolbar) {
  background: rgba(30, 33, 44, 0.72);
  border-color: rgba(255, 255, 255, 0.08);
}
.plaza-segment {
  background: rgba(120, 120, 128, 0.12);
}
:global(.dark .plaza-segment) {
  background: rgba(120, 120, 128, 0.24);
}
.plaza-segment-thumb {
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 2px;
  width: calc(50% - 2px);
  border-radius: 0.625rem;
  background: #fff;
  box-shadow:
    0 1px 2px rgba(15, 23, 42, 0.12),
    0 2px 8px rgba(15, 23, 42, 0.08);
  transition: transform 200ms cubic-bezier(0.23, 1, 0.32, 1);
}
:global(.dark .plaza-segment-thumb) {
  background: rgb(55 60 74);
}
.plaza-segment-btn {
  position: relative;
  z-index: 1;
  padding: 0.5rem 0.875rem;
  transition: color 160ms ease-out, transform 120ms ease-out;
}
.plaza-segment-btn:active {
  transform: scale(0.94);
}
.plaza-card {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(15, 23, 42, 0.06);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  transition:
    transform 220ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 220ms cubic-bezier(0.23, 1, 0.32, 1);
  animation: plaza-enter 360ms cubic-bezier(0.23, 1, 0.32, 1) both;
  animation-delay: var(--enter-delay, 0ms);
}
.plaza-card:hover {
  transform: translateY(-2px);
  box-shadow:
    0 2px 4px rgba(15, 23, 42, 0.04),
    0 12px 32px rgba(15, 23, 42, 0.1);
}
:global(.dark .plaza-card) {
  background: rgba(38, 42, 54, 0.85);
  border-color: rgba(255, 255, 255, 0.07);
}
:global(.dark .plaza-card):hover {
  box-shadow:
    0 2px 4px rgba(0, 0, 0, 0.2),
    0 12px 32px rgba(0, 0, 0, 0.35);
}
@keyframes plaza-enter {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.98);
  }
}
@media (prefers-reduced-motion: reduce) {
  .plaza-card {
    animation: none;
  }
  .plaza-card,
  .plaza-segment-thumb,
  .plaza-segment-btn {
    transition: none;
  }
}
@media (prefers-reduced-transparency: reduce) {
  .plaza-toolbar,
  .plaza-count-pill {
    backdrop-filter: none;
    background: #fff;
  }
}
</style>
