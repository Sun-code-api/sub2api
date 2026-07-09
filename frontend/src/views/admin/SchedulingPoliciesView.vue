<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="inline-flex rounded-lg bg-gray-100 p-0.5 dark:bg-gray-800">
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="activeTab === 'policies' ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'"
              @click="activeTab = 'policies'"
            >
              {{ t('admin.schedulingPolicies.tabPolicies') }}
            </button>
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="activeTab === 'actions' ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'"
              @click="switchToActions()"
            >
              {{ t('admin.schedulingPolicies.tabActions') }}
            </button>
          </div>

          <template v-if="activeTab === 'policies'">
            <SearchInput
              v-model="searchQuery"
              :placeholder="t('admin.schedulingPolicies.searchPlaceholder')"
              class="w-64"
              @input="handleSearch"
            />
            <div class="ml-auto">
              <button
                type="button"
                class="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3.5 py-2 text-sm font-medium text-white hover:bg-blue-700"
                @click="openCreateDialog"
              >
                <Icon name="plus" size="sm" />
                {{ t('admin.schedulingPolicies.createButton') }}
              </button>
            </div>
          </template>
          <template v-else>
            <select
              v-model="actionsPolicyFilter"
              class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
              @change="reloadActions"
            >
              <option :value="0">{{ t('admin.schedulingPolicies.allPolicies') }}</option>
              <option v-for="p in policies" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </template>
        </div>
      </template>

      <template #table>
        <DataTable v-if="activeTab === 'policies'" :columns="policyColumns" :data="policies" :loading="loading">
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>
          <template #cell-monitor_id="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ monitorName(row.monitor_id) }}</span>
          </template>
          <template #cell-accounts="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.accountsCount', { count: row.account_ids.length }) }}</span>
          </template>
          <template #cell-condition="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ conditionSummary(row) }}</span>
          </template>
          <template #cell-action="{ row }">
            <span
              class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
              :class="row.action_type === 'pause' ? 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
            >
              {{ actionSummary(row) }}
            </span>
          </template>
          <template #cell-enabled="{ row }">
            <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button type="button" class="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400" @click="openEditDialog(row)">
                {{ t('common.edit') }}
              </button>
              <button type="button" class="text-sm text-red-600 hover:text-red-700 dark:text-red-400" @click="handleDelete(row)">
                {{ t('common.delete') }}
              </button>
            </div>
          </template>
          <template #empty>
            <EmptyState
              :title="t('admin.schedulingPolicies.noPoliciesYet')"
              :description="t('admin.schedulingPolicies.createFirstPolicy')"
              :action-text="t('admin.schedulingPolicies.createButton')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>

        <DataTable v-else :columns="actionColumns" :data="actions" :loading="actionsLoading">
          <template #cell-policy_id="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ policyName(row.policy_id) }}</span>
          </template>
          <template #cell-account_id="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">#{{ row.account_id }}</span>
          </template>
          <template #cell-action="{ row }">
            <span
              class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
              :class="actionBadgeClass(row.action)"
            >
              {{ t(`admin.schedulingPolicies.actionKinds.${row.action}`) }}
            </span>
          </template>
          <template #cell-reason="{ row }">
            <span class="block max-w-md truncate text-sm text-gray-600 dark:text-gray-400" :title="row.reason">{{ row.reason }}</span>
          </template>
          <template #cell-restored="{ row }">
            <span class="text-sm" :class="row.restored ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'">
              {{ row.restored ? t('admin.schedulingPolicies.restored') : t('admin.schedulingPolicies.active') }}
            </span>
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-400">{{ formatTime(row.created_at) }}</span>
          </template>
          <template #empty>
            <EmptyState :title="t('admin.schedulingPolicies.noActionsYet')" :description="t('admin.schedulingPolicies.noActionsHint')" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="activeTab === 'policies' && pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
        <Pagination
          v-else-if="activeTab === 'actions' && actionsPagination.total > 0"
          :page="actionsPagination.page"
          :total="actionsPagination.total"
          :page-size="actionsPagination.page_size"
          @update:page="onActionsPageChange"
          @update:pageSize="onActionsPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showDialog"
      :title="editing ? t('admin.schedulingPolicies.editTitle') : t('admin.schedulingPolicies.createTitle')"
      width="wide"
      @close="closeDialog"
    >
      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.name') }}</label>
            <input
              v-model="form.name"
              type="text"
              required
              maxlength="100"
              class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
              :placeholder="t('admin.schedulingPolicies.form.namePlaceholder')"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.monitor') }}</label>
            <select
              v-model.number="form.monitor_id"
              required
              class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
            >
              <option :value="0" disabled>{{ t('admin.schedulingPolicies.form.monitorPlaceholder') }}</option>
              <option v-for="m in monitors" :key="m.id" :value="m.id">{{ m.name }} ({{ m.primary_model }})</option>
            </select>
          </div>
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.accounts') }}</label>
          <div class="max-h-40 space-y-1 overflow-y-auto rounded-lg border border-gray-300 p-2 dark:border-gray-600">
            <label v-for="a in accounts" :key="a.id" class="flex items-center gap-2 rounded px-1 py-0.5 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800">
              <input v-model="form.account_ids" type="checkbox" :value="a.id" class="rounded border-gray-300" />
              <span>#{{ a.id }} {{ a.name }}</span>
              <span class="ml-auto text-xs text-gray-400">{{ a.platform }}</span>
            </label>
            <p v-if="accounts.length === 0" class="px-1 py-2 text-sm text-gray-400">{{ t('admin.schedulingPolicies.form.noAccounts') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.consecutiveFailures') }}</label>
            <input v-model.number="form.trigger_consecutive_failures" type="number" min="1" max="100" required class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.latencyThreshold') }}</label>
            <input v-model.number="form.trigger_latency_ms" type="number" min="0" max="600000" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingPolicies.form.latencyThresholdHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.actionType') }}</label>
            <select v-model="form.action_type" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100">
              <option value="pause">{{ t('admin.schedulingPolicies.actionKinds.pause') }}</option>
              <option value="deprioritize">{{ t('admin.schedulingPolicies.actionKinds.deprioritize') }}</option>
            </select>
          </div>
          <div v-if="form.action_type === 'pause'">
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.pauseMinutes') }}</label>
            <input v-model.number="form.pause_minutes" type="number" min="0" max="10080" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingPolicies.form.pauseMinutesHint') }}</p>
          </div>
          <div v-else>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.priorityDelta') }}</label>
            <input v-model.number="form.priority_delta" type="number" min="1" max="1000" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingPolicies.form.priorityDeltaHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.recoverSuccesses') }}</label>
            <input v-model.number="form.recover_consecutive_successes" type="number" min="0" max="100" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingPolicies.form.recoverSuccessesHint') }}</p>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.schedulingPolicies.form.cooldownMinutes') }}</label>
            <input v-model.number="form.cooldown_minutes" type="number" min="0" max="1440" class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingPolicies.form.cooldownMinutesHint') }}</p>
          </div>
        </div>

        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300" />
          {{ t('admin.schedulingPolicies.form.enabled') }}
        </label>

        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 dark:border-gray-600 dark:text-gray-300" @click="closeDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" :disabled="saving" class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-60">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </form>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type {
  SchedulingPolicy,
  SchedulingPolicyParams,
  SchedulingAction,
  SchedulingActionKind,
} from '@/api/admin/schedulingPolicies'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
const appStore = useAppStore()

const activeTab = ref<'policies' | 'actions'>('policies')

const policies = ref<SchedulingPolicy[]>([])
const loading = ref(false)
const searchQuery = ref('')
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })

const actions = ref<SchedulingAction[]>([])
const actionsLoading = ref(false)
const actionsPolicyFilter = ref(0)
const actionsPagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })

const monitors = ref<ChannelMonitor[]>([])
const accounts = ref<Account[]>([])

const showDialog = ref(false)
const editing = ref<SchedulingPolicy | null>(null)
const saving = ref(false)
const showDeleteDialog = ref(false)
const deleting = ref<SchedulingPolicy | null>(null)

let searchTimeout: ReturnType<typeof setTimeout> | null = null

const emptyForm = (): SchedulingPolicyParams => ({
  name: '',
  enabled: true,
  monitor_id: 0,
  account_ids: [],
  trigger_consecutive_failures: 3,
  trigger_latency_ms: 0,
  action_type: 'pause',
  pause_minutes: 0,
  priority_delta: 10,
  recover_consecutive_successes: 3,
  cooldown_minutes: 10,
})

const form = reactive<SchedulingPolicyParams>(emptyForm())

const policyColumns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.schedulingPolicies.columns.name'), sortable: false },
  { key: 'monitor_id', label: t('admin.schedulingPolicies.columns.monitor'), sortable: false },
  { key: 'accounts', label: t('admin.schedulingPolicies.columns.accounts'), sortable: false },
  { key: 'condition', label: t('admin.schedulingPolicies.columns.condition'), sortable: false },
  { key: 'action', label: t('admin.schedulingPolicies.columns.action'), sortable: false },
  { key: 'enabled', label: t('admin.schedulingPolicies.columns.enabled'), sortable: false },
  { key: 'actions', label: t('admin.schedulingPolicies.columns.actions'), sortable: false },
])

const actionColumns = computed<Column[]>(() => [
  { key: 'policy_id', label: t('admin.schedulingPolicies.columns.policy'), sortable: false },
  { key: 'account_id', label: t('admin.schedulingPolicies.columns.account'), sortable: false },
  { key: 'action', label: t('admin.schedulingPolicies.columns.action'), sortable: false },
  { key: 'reason', label: t('admin.schedulingPolicies.columns.reason'), sortable: false },
  { key: 'restored', label: t('admin.schedulingPolicies.columns.status'), sortable: false },
  { key: 'created_at', label: t('admin.schedulingPolicies.columns.time'), sortable: false },
])

const deleteConfirmMessage = computed(() =>
  t('admin.schedulingPolicies.deleteConfirm', { name: deleting.value?.name || '' })
)

function monitorName(id: number): string {
  return monitors.value.find((m) => m.id === id)?.name || `#${id}`
}

function policyName(id: number): string {
  return policies.value.find((p) => p.id === id)?.name || `#${id}`
}

function conditionSummary(p: SchedulingPolicy): string {
  const parts = [t('admin.schedulingPolicies.failuresSummary', { count: p.trigger_consecutive_failures })]
  if (p.trigger_latency_ms > 0) {
    parts.push(t('admin.schedulingPolicies.latencySummary', { ms: p.trigger_latency_ms }))
  }
  return parts.join(' / ')
}

function actionSummary(p: SchedulingPolicy): string {
  if (p.action_type === 'pause') {
    return p.pause_minutes > 0
      ? t('admin.schedulingPolicies.pauseSummary', { minutes: p.pause_minutes })
      : t('admin.schedulingPolicies.pauseUntilRecoverySummary')
  }
  return t('admin.schedulingPolicies.deprioritizeSummary', { delta: p.priority_delta })
}

function actionBadgeClass(kind: SchedulingActionKind): string {
  switch (kind) {
    case 'pause':
      return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'deprioritize':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    default:
      return 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  }
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

async function reload() {
  loading.value = true
  try {
    const res = await adminAPI.schedulingPolicies.list({
      page: pagination.page,
      page_size: pagination.page_size,
      search: searchQuery.value.trim() || undefined,
    })
    policies.value = res.items || []
    pagination.total = res.total
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.schedulingPolicies.loadError')))
  } finally {
    loading.value = false
  }
}

async function reloadActions() {
  actionsLoading.value = true
  try {
    const res = await adminAPI.schedulingPolicies.listActions({
      page: actionsPagination.page,
      page_size: actionsPagination.page_size,
      policy_id: actionsPolicyFilter.value || undefined,
    })
    actions.value = res.items || []
    actionsPagination.total = res.total
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.schedulingPolicies.loadError')))
  } finally {
    actionsLoading.value = false
  }
}

function switchToActions() {
  activeTab.value = 'actions'
  reloadActions()
}

async function loadOptions() {
  try {
    const [monitorRes, accountRes] = await Promise.all([
      adminAPI.channelMonitor.list({ page: 1, page_size: 100 }),
      adminAPI.accounts.list(1, 200, { lite: 'true' }),
    ])
    monitors.value = monitorRes.items || []
    accounts.value = accountRes.items || []
  } catch {
    // 下拉选项加载失败不阻塞主列表
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function onPageChange(page: number) {
  pagination.page = page
  reload()
}

function onPageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function onActionsPageChange(page: number) {
  actionsPagination.page = page
  reloadActions()
}

function onActionsPageSizeChange(size: number) {
  actionsPagination.page_size = size
  actionsPagination.page = 1
  reloadActions()
}

function openCreateDialog() {
  editing.value = null
  Object.assign(form, emptyForm())
  showDialog.value = true
}

function openEditDialog(row: SchedulingPolicy) {
  editing.value = row
  Object.assign(form, {
    name: row.name,
    enabled: row.enabled,
    monitor_id: row.monitor_id,
    account_ids: [...row.account_ids],
    trigger_consecutive_failures: row.trigger_consecutive_failures,
    trigger_latency_ms: row.trigger_latency_ms,
    action_type: row.action_type,
    pause_minutes: row.pause_minutes,
    priority_delta: row.priority_delta,
    recover_consecutive_successes: row.recover_consecutive_successes,
    cooldown_minutes: row.cooldown_minutes,
  })
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editing.value = null
}

async function handleSubmit() {
  if (!form.monitor_id) {
    appStore.showError(t('admin.schedulingPolicies.monitorRequired'))
    return
  }
  if (form.account_ids.length === 0) {
    appStore.showError(t('admin.schedulingPolicies.accountsRequired'))
    return
  }
  saving.value = true
  try {
    const payload: SchedulingPolicyParams = { ...form, priority_delta: form.priority_delta || 10 }
    if (editing.value) {
      await adminAPI.schedulingPolicies.update(editing.value.id, payload)
      appStore.showSuccess(t('admin.schedulingPolicies.updateSuccess'))
    } else {
      await adminAPI.schedulingPolicies.create(payload)
      appStore.showSuccess(t('admin.schedulingPolicies.createSuccess'))
    }
    closeDialog()
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(row: SchedulingPolicy) {
  const next = !row.enabled
  try {
    await adminAPI.schedulingPolicies.update(row.id, {
      name: row.name,
      enabled: next,
      monitor_id: row.monitor_id,
      account_ids: row.account_ids,
      trigger_consecutive_failures: row.trigger_consecutive_failures,
      trigger_latency_ms: row.trigger_latency_ms,
      action_type: row.action_type,
      pause_minutes: row.pause_minutes,
      priority_delta: row.priority_delta,
      recover_consecutive_successes: row.recover_consecutive_successes,
      cooldown_minutes: row.cooldown_minutes,
    })
    row.enabled = next
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

function handleDelete(row: SchedulingPolicy) {
  deleting.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  try {
    await adminAPI.schedulingPolicies.remove(deleting.value.id)
    appStore.showSuccess(t('admin.schedulingPolicies.deleteSuccess'))
    showDeleteDialog.value = false
    deleting.value = null
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(() => {
  reload()
  loadOptions()
})
onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
})
</script>
