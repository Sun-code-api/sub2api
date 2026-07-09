<template>
  <!-- 落地页模型价格预览：调用匿名版模型广场接口，无数据时整块隐藏 -->
  <div v-if="models.length > 0" class="mb-16">
    <div class="mb-8 text-center">
      <h2 class="mb-3 text-2xl font-bold text-gray-900 dark:text-white">
        {{ t('home.pricingPreview.title') }}
      </h2>
      <p class="text-sm text-gray-600 dark:text-dark-400">
        {{ t('home.pricingPreview.description') }}
      </p>
    </div>

    <div
      class="overflow-hidden rounded-2xl border border-gray-200/50 bg-white/70 backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/70"
    >
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700/60 dark:text-dark-400">
            <th class="px-5 py-3 font-medium">{{ t('home.pricingPreview.model') }}</th>
            <th class="hidden px-5 py-3 font-medium sm:table-cell">{{ t('home.pricingPreview.platform') }}</th>
            <th class="px-5 py-3 text-right font-medium">{{ t('home.pricingPreview.input') }}</th>
            <th class="px-5 py-3 text-right font-medium">{{ t('home.pricingPreview.output') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="m in previewModels"
            :key="m.platform + '/' + m.name"
            class="border-b border-gray-50 last:border-0 dark:border-dark-700/40"
          >
            <td class="break-all px-5 py-3 font-medium text-gray-900 dark:text-white">{{ m.name }}</td>
            <td class="hidden px-5 py-3 text-gray-500 dark:text-dark-400 sm:table-cell">
              {{ platformLabel(m.platform) }}
            </td>
            <td class="px-5 py-3 text-right font-mono text-gray-900 dark:text-white">
              {{ price(m.pricing?.input_price) }}
            </td>
            <td class="px-5 py-3 text-right font-mono text-gray-900 dark:text-white">
              {{ price(m.pricing?.output_price) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="mt-6 text-center">
      <router-link
        to="/models"
        class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
      >
        {{ t('home.pricingPreview.viewAll') }}
        <Icon name="arrowRight" size="sm" :stroke-width="2" />
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import modelPlazaAPI, { type ModelPlazaEntry } from '@/api/modelPlaza'

const { t } = useI18n()

const models = ref<ModelPlazaEntry[]>([])

const PREVIEW_LIMIT = 8

// 每个平台轮流取一个，保证预览覆盖多平台而不是被单一平台刷屏
const previewModels = computed(() => {
  const byPlatform = new Map<string, ModelPlazaEntry[]>()
  for (const m of models.value) {
    if (!m.pricing || m.pricing.input_price === null) continue
    const list = byPlatform.get(m.platform) || []
    list.push(m)
    byPlatform.set(m.platform, list)
  }
  const out: ModelPlazaEntry[] = []
  let idx = 0
  while (out.length < PREVIEW_LIMIT) {
    let added = false
    for (const list of byPlatform.values()) {
      if (idx < list.length && out.length < PREVIEW_LIMIT) {
        out.push(list[idx])
        added = true
      }
    }
    if (!added) break
    idx++
  }
  return out
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

function price(v: number | null | undefined): string {
  if (v === null || v === undefined) return '—'
  const mtok = v * 1_000_000
  return `$${mtok >= 100 ? mtok.toFixed(0) : mtok.toPrecision(3)}`
}

onMounted(async () => {
  try {
    models.value = await modelPlazaAPI.getPublicModelPlaza()
  } catch {
    // 匿名预览失败静默降级：整块不渲染
    models.value = []
  }
})
</script>
