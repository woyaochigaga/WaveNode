<template>
  <AppLayout>
    <TablePageLayout>
      <!-- Filters -->
      <template #filters>
        <div class="card p-3 sm:p-4">
          <div class="grid grid-cols-1 items-end gap-3 md:grid-cols-2 xl:grid-cols-[minmax(300px,1fr)_150px_190px_auto]">
            <div>
              <label class="input-label">{{ t('admin.audit.filters.q') }}</label>
              <div class="relative">
                <Icon name="search" size="md" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  v-model.trim="filters.q"
                  type="text"
                  class="input pl-10"
                  :placeholder="t('admin.audit.filters.qPlaceholder')"
                  @keyup.enter="search"
                />
              </div>
            </div>

            <div>
              <label class="input-label">{{ t('admin.audit.filters.result') }}</label>
              <Select v-model="filters.success" :options="resultOptions" @change="search" />
            </div>

            <div>
              <label class="input-label">{{ t('admin.dashboard.timeRange') }}</label>
              <Select
                :model-value="timeRange"
                :options="timeRangeOptions"
                @update:model-value="handleTimeRangeChange"
              />
            </div>

            <div class="flex flex-wrap items-center gap-2 xl:justify-end">
              <button
                type="button"
                class="btn btn-secondary"
                :aria-expanded="advancedFiltersOpen"
                @click="advancedFiltersOpen = !advancedFiltersOpen"
              >
                <Icon name="filter" size="sm" class="mr-1.5" />
                {{ t('admin.audit.filters.advanced') }}
                <span v-if="advancedFilterCount" class="ml-1 rounded-full bg-primary-100 px-1.5 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
                  {{ advancedFilterCount }}
                </span>
              </button>
              <button type="button" class="btn btn-primary" :disabled="loading" @click="search">
                <Icon name="search" size="sm" class="mr-1.5" />
                {{ t('common.search') }}
              </button>
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="resetFilters">
                {{ t('common.reset') }}
              </button>
              <button type="button" class="btn btn-danger" :title="t('admin.audit.clearAll')" @click="openClearDialog">
                <Icon name="trash" size="sm" />
                <span class="sr-only">{{ t('admin.audit.clearAll') }}</span>
              </button>
            </div>
          </div>

          <div v-if="advancedFiltersOpen" class="mt-3 grid grid-cols-1 gap-3 border-t border-gray-200 pt-3 sm:grid-cols-2 lg:grid-cols-5 dark:border-dark-700">
            <div>
              <label class="input-label">{{ t('admin.audit.filters.actorEmail') }}</label>
              <input v-model.trim="filters.actor_email" type="text" class="input" @keyup.enter="search" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.audit.filters.action') }}</label>
              <input v-model.trim="filters.action" type="text" class="input" @keyup.enter="search" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.audit.filters.clientIp') }}</label>
              <input v-model.trim="filters.client_ip" type="text" class="input" @keyup.enter="search" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.audit.filters.method') }}</label>
              <Select v-model="filters.method" :options="methodOptions" @change="search" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.audit.filters.authMethod') }}</label>
              <Select v-model="filters.auth_method" :options="authMethodOptions" @change="search" />
            </div>
          </div>
        </div>
      </template>

      <!-- Table -->
      <template #table>
        <DataTable :columns="columns" :data="logs" :loading="loading" row-key="id">
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatTime(value) }}</span>
          </template>

          <template #cell-actor="{ row }">
            <div class="min-w-0 max-w-[220px]">
              <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.actor_email">
                {{ row.actor_email || '—' }}
              </div>
              <div class="mt-0.5 truncate text-xs text-gray-400">
                {{ row.actor_role }}<span v-if="row.auth_method"> · {{ authMethodLabel(row.auth_method) }}</span>
              </div>
            </div>
          </template>

          <template #cell-action="{ row }">
            <div class="min-w-0 max-w-xs">
              <div class="truncate font-mono text-sm text-gray-800 dark:text-gray-200" :title="row.action">
                {{ row.action }}
              </div>
              <div class="mt-0.5 truncate font-mono text-xs text-gray-400" :title="`${row.method} ${row.path}`">
                {{ row.method }} {{ row.path }}
              </div>
            </div>
          </template>

          <template #cell-status_code="{ row }">
            <span :class="statusBadgeClass(row.status_code)">
              <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(row.status_code)"></span>
              {{ row.status_code }}
            </span>
          </template>

          <template #cell-latency_ms="{ value }">
            <span class="whitespace-nowrap text-gray-500 dark:text-gray-400">{{ value }} ms</span>
          </template>

          <template #cell-client_ip="{ value }">
            <span class="whitespace-nowrap font-mono text-gray-600 dark:text-gray-300">{{ value || '—' }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="inline-flex items-center gap-1 font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              @click="openDetail(row.id)"
            >
              <Icon name="eye" size="sm" />
              {{ t('admin.audit.columns.detail') }}
            </button>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-8">
              <Icon name="shield" size="xl" class="mb-4 h-12 w-12 text-gray-300 dark:text-dark-600" />
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.audit.empty') }}</p>
            </div>
          </template>
        </DataTable>
      </template>

      <!-- Pagination -->
      <template #pagination>
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Detail dialog -->
    <BaseDialog
      :show="detailVisible"
      :title="t('admin.audit.detail.title')"
      width="extra-wide"
      :close-on-click-outside="true"
      @close="detailVisible = false"
    >
      <div v-if="detailLoading" class="flex items-center justify-center py-16">
        <div class="flex flex-col items-center gap-3">
          <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
          <div class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
        </div>
      </div>

      <div v-else-if="detail" class="min-w-0 space-y-4 py-2">
        <!-- Hero: action + result at a glance -->
        <div class="min-w-0 rounded-lg border border-gray-200 bg-gray-50/60 p-5 dark:border-dark-700 dark:bg-dark-900/60">
          <div class="flex flex-wrap items-center gap-3">
            <span :class="statusBadgeClass(detail.status_code)">
              <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(detail.status_code)"></span>
              {{ detail.status_code }} {{ statusText(detail.status_code) }}
            </span>
            <span class="break-all font-mono text-base font-semibold text-gray-900 dark:text-white">
              {{ detail.action }}
            </span>
          </div>

          <div class="mt-3 flex items-center gap-2 rounded-lg bg-white px-3 py-2 ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-600">
            <span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[11px] font-bold text-gray-700 dark:bg-dark-700 dark:text-gray-200">
              {{ detail.method }}
            </span>
            <span class="min-w-0 break-all font-mono text-xs text-gray-600 dark:text-gray-300">{{ detail.path }}</span>
          </div>

          <div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span class="inline-flex items-center gap-1.5">
              <Icon name="clock" size="xs" />
              {{ formatTime(detail.created_at) }}
            </span>
            <span>{{ t('admin.audit.detail.latency') }} {{ detail.latency_ms }} ms</span>
            <span v-if="detail.request_id" class="inline-flex items-center gap-1">
              {{ t('admin.audit.detail.requestId') }}
              <span class="break-all font-mono">{{ detail.request_id }}</span>
            </span>
          </div>
        </div>

        <div class="grid grid-cols-3 rounded-lg bg-gray-100 p-1 dark:bg-dark-800" role="tablist">
          <button
            v-for="tab in detailTabs"
            :key="tab.value"
            type="button"
            role="tab"
            :aria-selected="detailTab === tab.value"
            :class="[
              'h-9 rounded-md px-3 text-sm font-medium transition-colors',
              detailTab === tab.value
                ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white'
                : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
            @click="detailTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-if="detailTab === 'overview'" class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.columns.actor') }}</div>
              <div class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ detail.actor_email || '—' }}</div>
              <div class="mt-0.5 text-xs text-gray-400">{{ detail.actor_role }}</div>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.filters.authMethod') }}</div>
              <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ authMethodLabel(detail.auth_method) || '—' }}</div>
              <div v-if="detail.credential_masked" class="mt-0.5 break-all font-mono text-xs text-gray-400">{{ detail.credential_masked }}</div>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.columns.clientIp') }}</div>
              <div class="mt-1 break-all font-mono text-sm font-medium text-gray-900 dark:text-white">{{ detail.client_ip || '—' }}</div>
            </div>
          </div>
          <PayloadBlock :title="t('admin.audit.detail.userAgent')" :content="detail.user_agent || '—'" />
          <PayloadBlock
            v-if="detailMetadata"
            :title="t('admin.audit.detail.extra')"
            :content="JSON.stringify(detailMetadata, null, 2)"
          />
        </div>

        <div v-else-if="detailTab === 'request'" class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <PayloadBlock
            :title="t('admin.audit.detail.pathParams')"
            :content="formatStructuredValue(detail.extra?.params)"
            :empty-text="t('admin.audit.detail.noPathParams')"
          />
          <PayloadBlock
            :title="t('admin.audit.detail.queryParams')"
            :content="formatQuery(detail.extra?.query)"
            :empty-text="t('admin.audit.detail.noQueryParams')"
          />
          <div class="lg:col-span-2">
            <PayloadBlock
              :title="t('admin.audit.detail.requestBody')"
              :content="detail.request_body ? prettyBody(detail.request_body) : ''"
              :content-type="detail.request_content_type"
              :empty-text="t('admin.audit.detail.noRequestBody')"
              tall
            />
          </div>
        </div>

        <div v-else class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.columns.result') }}</div>
              <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ detail.status_code }} {{ statusText(detail.status_code) }}</div>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.detail.latency') }}</div>
              <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ detail.latency_ms }} ms</div>
            </div>
            <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900">
              <div class="text-xs font-bold uppercase text-gray-400">{{ t('admin.audit.detail.responseType') }}</div>
              <div class="mt-1 break-all font-mono text-xs text-gray-700 dark:text-gray-300">{{ detail.response_content_type || '—' }}</div>
            </div>
          </div>
          <PayloadBlock
            :title="t('admin.audit.detail.responseBody')"
            :content="detail.response_body ? prettyBody(detail.response_body) : ''"
            :content-type="detail.response_content_type"
            :empty-text="t('admin.audit.detail.noResponseBody')"
            tall
          />
        </div>
      </div>
    </BaseDialog>

    <!-- Custom time range dialog (与 /admin/ops 时间下拉一致的自定义范围，支持时分) -->
    <BaseDialog
      :show="showCustomTimeRangeDialog"
      :title="t('admin.ops.timeRange.custom')"
      width="narrow"
      @close="handleCustomTimeRangeCancel"
    >
      <div class="space-y-4 py-2">
        <div>
          <label class="input-label">{{ t('admin.ops.customTimeRange.startTime') }}</label>
          <input v-model="customStartTimeInput" type="datetime-local" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.ops.customTimeRange.endTime') }}</label>
          <input v-model="customEndTimeInput" type="datetime-local" class="input" />
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="handleCustomTimeRangeCancel">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="!customStartTimeInput || !customEndTimeInput"
          @click="handleCustomTimeRangeConfirm"
        >
          {{ t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>

    <!-- Clear confirmation → step-up TOTP -->
    <ConfirmDialog
      :show="clearConfirmVisible"
      :title="t('admin.audit.clearConfirm.title')"
      :message="t('admin.audit.clearConfirm.message')"
      :confirm-text="t('admin.audit.clearAll')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="onClearConfirmed"
      @cancel="clearConfirmVisible = false"
    />

    <!-- TOTP prompt for the clear operation -->
    <BaseDialog
      :show="clearTotpVisible"
      :title="t('admin.audit.clearConfirm.totpTitle')"
      width="narrow"
      :z-index="60"
      @close="cancelClearTotp"
    >
      <div class="py-2">
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.audit.clearConfirm.totpHint') }}</p>
        <input
          v-model.trim="clearTotpCode"
          type="text"
          inputmode="numeric"
          maxlength="6"
          autocomplete="one-time-code"
          class="input mt-4 text-center text-lg tracking-[0.5em]"
          placeholder="••••••"
          @keyup.enter="submitClear"
        />
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="clearing" @click="cancelClearTotp">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-danger"
          :disabled="clearing || clearTotpCode.length !== 6"
          @click="submitClear"
        >
          {{ clearing ? t('common.loading') : t('admin.audit.clearAll') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type AuditLog } from '@/api/admin'
import { totpAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PayloadBlock from '@/components/admin/audit/AuditPayloadBlock.vue'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const advancedFiltersOpen = ref(false)

const filters = reactive({
  q: '',
  actor_email: '',
  action: '',
  client_ip: '',
  method: '',
  auth_method: '',
  success: ''
})

// 高级条件保持折叠，但用数量徽标提醒当前仍在生效的筛选项。
const advancedFilterCount = computed(() =>
  [filters.actor_email, filters.action, filters.client_ip, filters.method, filters.auth_method]
    .filter(Boolean).length
)

// 时间范围：预设窗口（同 /admin/ops 时间下拉）+ 自定义起止（datetime-local，支持时分）
const timeRange = ref('')
const customStartTime = ref('')
const customEndTime = ref('')
const showCustomTimeRangeDialog = ref(false)
const customStartTimeInput = ref('')
const customEndTimeInput = ref('')

const TIME_RANGE_MINUTES: Record<string, number> = {
  '30m': 30,
  '1h': 60,
  '6h': 6 * 60,
  '24h': 24 * 60,
  '7d': 7 * 24 * 60,
  '30d': 30 * 24 * 60
}

const timeRangeOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: '30m', label: t('admin.ops.timeRange.30m') },
  { value: '1h', label: t('admin.ops.timeRange.1h') },
  { value: '6h', label: t('admin.ops.timeRange.6h') },
  { value: '24h', label: t('admin.ops.timeRange.24h') },
  { value: '7d', label: t('admin.ops.timeRange.7d') },
  { value: '30d', label: t('admin.ops.timeRange.30d') },
  {
    value: 'custom',
    label:
      timeRange.value === 'custom' && customStartTime.value && customEndTime.value
        ? `${t('admin.ops.timeRange.custom')} (${formatCustomTimeRangeLabel(customStartTime.value, customEndTime.value)})`
        : t('admin.ops.timeRange.custom')
  }
])

function formatCustomTimeRangeLabel(startTime: string, endTime: string): string {
  const fmt = (raw: string) => {
    const d = new Date(raw)
    if (Number.isNaN(d.getTime())) return raw
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  }
  return `${fmt(startTime)} ~ ${fmt(endTime)}`
}

function toDatetimeLocal(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function handleTimeRangeChange(val: string | number | boolean | null) {
  const value = String(val ?? '')
  if (value === 'custom') {
    // 预填：已有自定义值沿用，否则默认最近1小时（本地时区）
    const now = new Date()
    customStartTimeInput.value = customStartTime.value || toDatetimeLocal(new Date(now.getTime() - 60 * 60 * 1000))
    customEndTimeInput.value = customEndTime.value || toDatetimeLocal(now)
    showCustomTimeRangeDialog.value = true
    return
  }
  timeRange.value = value
  search()
}

function handleCustomTimeRangeConfirm() {
  if (!customStartTimeInput.value || !customEndTimeInput.value) return
  customStartTime.value = customStartTimeInput.value
  customEndTime.value = customEndTimeInput.value
  timeRange.value = 'custom'
  showCustomTimeRangeDialog.value = false
  search()
}

function handleCustomTimeRangeCancel() {
  // 未确认不改变当前时间范围；Select 是受控组件，展示值保持不变。
  showCustomTimeRangeDialog.value = false
}

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.audit.columns.time') },
  { key: 'actor', label: t('admin.audit.columns.actor') },
  { key: 'action', label: t('admin.audit.columns.action') },
  { key: 'status_code', label: t('admin.audit.columns.result') },
  { key: 'latency_ms', label: t('admin.audit.detail.latency') },
  { key: 'client_ip', label: t('admin.audit.columns.clientIp') },
  { key: 'actions', label: t('common.actions') }
])

const methodOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'GET', label: 'GET' }
])

const authMethodOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'jwt', label: 'JWT' },
  { value: 'admin_api_key', label: 'Admin API Key' }
])

const resultOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'true', label: t('admin.audit.filters.resultSuccess') },
  { value: 'false', label: t('admin.audit.filters.resultFailure') }
])

function authMethodLabel(method: string): string {
  const found = authMethodOptions.value.find((o) => o.value === method)
  return found && found.value ? found.label : method
}

function toRFC3339(local: string): string | undefined {
  if (!local) return undefined
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toISOString()
}

function buildTimeRangeQuery(): { start_time?: string; end_time?: string } {
  if (timeRange.value === 'custom') {
    return {
      start_time: toRFC3339(customStartTime.value),
      end_time: toRFC3339(customEndTime.value)
    }
  }
  const minutes = TIME_RANGE_MINUTES[timeRange.value]
  if (!minutes) return {}
  return { start_time: new Date(Date.now() - minutes * 60 * 1000).toISOString() }
}

function buildQuery() {
  return {
    page: page.value,
    page_size: pageSize.value,
    q: filters.q || undefined,
    actor_email: filters.actor_email || undefined,
    action: filters.action || undefined,
    client_ip: filters.client_ip || undefined,
    method: filters.method || undefined,
    auth_method: filters.auth_method || undefined,
    success: filters.success || undefined,
    ...buildTimeRangeQuery()
  }
}

async function fetchLogs() {
  loading.value = true
  try {
    const res = await adminAPI.audit.list(buildQuery())
    logs.value = res.items
    total.value = res.total
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  fetchLogs()
}

function resetFilters() {
  filters.q = ''
  filters.actor_email = ''
  filters.action = ''
  filters.client_ip = ''
  filters.method = ''
  filters.auth_method = ''
  filters.success = ''
  timeRange.value = ''
  customStartTime.value = ''
  customEndTime.value = ''
  search()
}

function onPageChange(p: number) {
  page.value = p
  fetchLogs()
}

function onPageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchLogs()
}

// Detail dialog
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AuditLog | null>(null)
const detailTab = ref<'overview' | 'request' | 'response'>('overview')
const detailTabs = computed(() => [
  { value: 'overview' as const, label: t('admin.audit.detail.tabs.overview') },
  { value: 'request' as const, label: t('admin.audit.detail.tabs.request') },
  { value: 'response' as const, label: t('admin.audit.detail.tabs.response') }
])

const detailMetadata = computed(() => {
  if (!detail.value?.extra) return null
  const { params: _params, query: _query, ...metadata } = detail.value.extra
  return Object.keys(metadata).length ? metadata : null
})

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  detailTab.value = 'overview'
  try {
    detail.value = await adminAPI.audit.get(id)
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

function prettyBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2)
  } catch {
    return body
  }
}

function formatStructuredValue(value: unknown): string {
  if (value == null || value === '') return ''
  return typeof value === 'string' ? value : JSON.stringify(value, null, 2)
}

// 查询参数转为逐字段 JSON，重复参数保留为数组，便于定位真实输入。
function formatQuery(value: unknown): string {
  if (typeof value !== 'string' || !value.trim()) return ''
  const params = new URLSearchParams(value)
  const result: Record<string, string | string[]> = {}
  for (const [key, item] of params.entries()) {
    const current = result[key]
    if (current == null) result[key] = item
    else if (Array.isArray(current)) current.push(item)
    else result[key] = [current, item]
  }
  return Object.keys(result).length ? JSON.stringify(result, null, 2) : value
}

// Clear-all flow: confirm → TOTP → clear
const clearConfirmVisible = ref(false)
const clearTotpVisible = ref(false)
const clearTotpCode = ref('')
const clearing = ref(false)
const checkingTotpStatus = ref(false)

// 与其他敏感操作一致：未启用 2FA 时直接提示去个人资料启用 TOTP，
// 而不是弹出一个无法完成的验证码输入框（后端会以 TOTP_NOT_SETUP 拒绝）。
async function openClearDialog() {
  if (checkingTotpStatus.value) return
  checkingTotpStatus.value = true
  try {
    const status = await totpAPI.getStatus()
    if (!status.enabled) {
      appStore.showError(t('stepUp.notEnabled'))
      return
    }
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
    return
  } finally {
    checkingTotpStatus.value = false
  }
  clearConfirmVisible.value = true
}

function onClearConfirmed() {
  clearConfirmVisible.value = false
  clearTotpCode.value = ''
  clearTotpVisible.value = true
}

function cancelClearTotp() {
  if (clearing.value) return
  clearTotpVisible.value = false
}

async function submitClear() {
  if (clearTotpCode.value.length !== 6) return
  clearing.value = true
  try {
    const res = await adminAPI.audit.clear(clearTotpCode.value)
    clearTotpVisible.value = false
    appStore.showSuccess(t('admin.audit.clearConfirm.success', { count: res.deleted }))
    search()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.clearConfirm.failed'))
    clearTotpCode.value = ''
  } finally {
    clearing.value = false
  }
}

// Helpers
function formatTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

function statusText(status: number): string {
  return status < 400 ? t('admin.audit.filters.resultSuccess') : t('admin.audit.filters.resultFailure')
}

function statusBadgeClass(status: number): string {
  const base = 'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold '
  if (status >= 500) return base + 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (status >= 400) return base + 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return base + 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
}

function statusDotClass(status: number): string {
  if (status >= 500) return 'bg-red-500'
  if (status >= 400) return 'bg-amber-500'
  return 'bg-green-500'
}

onMounted(fetchLogs)
</script>
