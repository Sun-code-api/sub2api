<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 渐变横幅 -->
      <div
        class="relative overflow-hidden rounded-2xl bg-gradient-to-r from-indigo-100 via-purple-100 to-pink-100 px-6 py-8 dark:from-indigo-950/60 dark:via-purple-950/60 dark:to-pink-950/60"
      >
        <div class="flex items-center gap-4">
          <div
            class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 shadow-lg shadow-purple-500/30"
          >
            <Icon name="sparkles" size="lg" class="text-white" />
          </div>
          <div>
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">
              {{ t('modelPlaza.title') }}
            </h1>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">
              {{ t('modelPlaza.bannerDescription') }}
            </p>
          </div>
        </div>
      </div>

      <!-- 工具栏 -->
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
          <div
            class="flex items-center overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600"
          >
            <button
              class="px-2.5 py-2 transition-colors"
              :class="
                viewMode === 'card'
                  ? 'bg-primary-500 text-white'
                  : 'bg-white text-gray-500 hover:bg-gray-50 dark:bg-dark-800 dark:text-dark-300 dark:hover:bg-dark-700'
              "
              :title="t('modelPlaza.cardView')"
              @click="viewMode = 'card'"
            >
              <Icon name="grid" size="sm" />
            </button>
            <button
              class="px-2.5 py-2 transition-colors"
              :class="
                viewMode === 'list'
                  ? 'bg-primary-500 text-white'
                  : 'bg-white text-gray-500 hover:bg-gray-50 dark:bg-dark-800 dark:text-dark-300 dark:hover:bg-dark-700'
              "
              :title="t('modelPlaza.listView')"
              @click="viewMode = 'list'"
            >
              <Icon name="menu" size="sm" />
            </button>
          </div>
          <span class="whitespace-nowrap text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelPlaza.modelsCount', { count: filteredModels.length }) }}
          </span>
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
          v-for="m in filteredModels"
          :key="m.platform + '/' + m.name"
          class="flex flex-col rounded-2xl border border-gray-200/70 bg-white p-4 transition-shadow hover:shadow-lg hover:shadow-primary-500/10 dark:border-dark-700/70 dark:bg-dark-800"
        >
          <!-- 提供商 + 状态 -->
          <div class="mb-1 flex items-center justify-between">
            <span class="text-xs text-gray-400 dark:text-dark-500">{{ m.platform }}</span>
            <span
              class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400"
            >
              <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
              {{ t('modelPlaza.available') }}
            </span>
          </div>
          <h3 class="mb-3 break-all text-sm font-semibold text-gray-900 dark:text-white">
            {{ m.name }}
          </h3>

          <!-- 分色价格块 -->
          <div v-if="m.pricing" class="mb-3 grid grid-cols-2 gap-2">
            <div
              v-for="block in priceBlocks(m.pricing)"
              :key="block.label"
              class="rounded-lg px-3 py-2"
              :class="block.class"
            >
              <div class="text-[10px]" :class="block.labelClass">{{ block.label }}</div>
              <div class="font-mono text-sm font-semibold text-gray-900 dark:text-white">
                {{ block.value }}
              </div>
              <div class="text-[10px] text-gray-400 dark:text-dark-500">{{ block.unit }}</div>
            </div>
          </div>
          <div v-else class="mb-3 text-xs text-gray-400 dark:text-dark-500">
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
        class="overflow-x-auto rounded-2xl border border-gray-200/70 bg-white dark:border-dark-700/70 dark:bg-dark-800"
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
              <td class="px-4 py-3 font-mono text-gray-900 dark:text-white">
                {{ m.pricing?.input_price != null ? perMTok(m.pricing.input_price) : '—' }}
              </td>
              <td class="px-4 py-3 font-mono text-gray-900 dark:text-white">
                {{ m.pricing?.output_price != null ? perMTok(m.pricing.output_price) : '—' }}
              </td>
              <td class="px-4 py-3 font-mono text-sky-600 dark:text-sky-400">
                {{ m.pricing?.cache_read_price != null ? perMTok(m.pricing.cache_read_price) : '—' }}
              </td>
              <td class="px-4 py-3 font-mono text-pink-600 dark:text-pink-400">
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
