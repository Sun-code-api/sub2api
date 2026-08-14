import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpencodeQuotaBadge from '../OpencodeQuotaBadge.vue'
import type { Account } from '@/types'

const { queryOpencodeQuota } = vi.hoisted(() => ({
  queryOpencodeQuota: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: { queryOpencodeQuota }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const account = {
  id: 165,
  platform: 'opencode',
  type: 'apikey',
  extra: {}
} as Account

describe('OpencodeQuotaBadge', () => {
  beforeEach(() => {
    queryOpencodeQuota.mockReset()
  })

  it('queries usage and emits merged extra', async () => {
    queryOpencodeQuota.mockResolvedValue({
      weekly: { percent: 42, reset_in_sec: 100, status: 'ok' },
      extra: {
        opencode_quota_weekly_pct: 42,
        opencode_quota_refreshed_at: '2026-08-14T08:00:00Z'
      }
    })
    const wrapper = mount(OpencodeQuotaBadge, { props: { account } })

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(queryOpencodeQuota).toHaveBeenCalledWith(165)
    expect(wrapper.emitted('account-updated')?.[0]?.[0]).toMatchObject({
      id: 165,
      extra: {
        opencode_quota_weekly_pct: 42,
        opencode_quota_refreshed_at: '2026-08-14T08:00:00Z'
      }
    })
  })
})
