<template>
  <BaseDialog :show="show" :title="operation === 'add' ? t('admin.users.deposit') : t('admin.users.withdraw')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="balance-form" @submit.prevent="handleBalanceSubmit" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100"><span class="text-lg font-medium text-primary-700">{{ user.email.charAt(0).toUpperCase() }}</span></div>
        <div class="flex-1">
          <p class="font-medium text-gray-900 dark:text-gray-100">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.users.currentBalance') }}: ${{ formatBalance(permanentBalance) }}</p>
          <p v-if="bonusBalance > 0" class="text-xs text-red-600 dark:text-red-400">{{ t('payment.rechargePackages.currentBonus', { amount: formatBalance(bonusBalance) }) }}</p>
          <p v-if="bonusExpiryText" class="text-[10px] text-red-500 dark:text-red-400">{{ bonusExpiryText }}</p>
        </div>
      </div>
      <div>
        <label class="input-label">{{ operation === 'add' ? t('admin.users.depositAmount') : t('admin.users.withdrawAmount') }}</label>
        <div class="relative flex gap-2">
          <div class="relative flex-1"><div class="absolute left-3 top-1/2 -translate-y-1/2 font-medium text-gray-500">$</div><input v-model.number="form.amount" type="number" step="any" min="0" required class="input pl-8" /></div>
          <button v-if="operation === 'subtract'" type="button" @click="fillAllBalance" class="btn btn-secondary whitespace-nowrap">{{ t('admin.users.withdrawAll') }}</button>
        </div>
      </div>
      <div v-if="operation === 'add'" class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.users.bonusAmount') }}</label>
          <div class="relative"><div class="absolute left-3 top-1/2 -translate-y-1/2 font-medium text-gray-500">$</div><input v-model.number="form.bonusAmount" type="number" step="any" min="0" class="input pl-8" /></div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.users.bonusValidityDays') }}</label>
          <div class="relative"><input v-model.number="form.bonusValidityDays" type="number" min="1" max="3650" :disabled="!form.bonusAmount" class="input pr-14" /><span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">{{ t('admin.users.days') }}</span></div>
        </div>
      </div>
      <p v-if="operation === 'add' && form.bonusAmount > 0" class="-mt-2 text-xs text-red-600 dark:text-red-400">{{ t('admin.users.bonusValidityHint') }}</p>
      <div><label class="input-label">{{ t('admin.users.notes') }}</label><textarea v-model="form.notes" rows="3" class="input"></textarea></div>
      <div v-if="form.amount > 0 || form.bonusAmount > 0" class="rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950"><div class="space-y-1 text-sm"><div class="flex items-center justify-between"><span class="text-gray-700 dark:text-gray-300">{{ t('admin.users.newBalance') }}:</span><span class="font-bold text-gray-900 dark:text-gray-100">${{ formatBalance(calculateNewBalance()) }}</span></div><div v-if="operation === 'add' && form.bonusAmount > 0" class="flex items-center justify-between text-red-600 dark:text-red-400"><span>{{ t('admin.users.bonusAmount') }}:</span><span class="font-semibold">+${{ formatBalance(form.bonusAmount) }}</span></div></div></div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="balance-form" :disabled="submitting || !canSubmit" class="btn" :class="operation === 'add' ? 'bg-emerald-600 text-white' : 'btn-danger'">{{ submitting ? t('common.saving') : t('common.confirm') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { formatDateTimeToMinute } from '@/utils/format'

const props = defineProps<{ show: boolean, user: AdminUser | null, operation: 'add' | 'subtract' }>()
const emit = defineEmits(['close', 'success']); const { t } = useI18n(); const appStore = useAppStore()

const submitting = ref(false); const form = reactive({ amount: 0, bonusAmount: 0, bonusValidityDays: 30, notes: '' })
const bonusBalance = ref(0)
const nearestBonusExpiry = ref<string | null>(null)
const nearestBonusExpiryAmount = ref(0)
const permanentBalance = computed(() => Math.max(0, Number(props.user?.balance || 0) - bonusBalance.value))
const canSubmit = computed(() => props.operation === 'add' ? form.amount > 0 || form.bonusAmount > 0 : form.amount > 0)
const bonusExpiryText = computed(() => {
  if (bonusBalance.value <= 0 || !nearestBonusExpiry.value || nearestBonusExpiryAmount.value <= 0) return ''
  return t('payment.rechargePackages.bonusExpiry', { amount: formatBalance(nearestBonusExpiryAmount.value), date: formatDateTimeToMinute(nearestBonusExpiry.value) })
})
watch(() => props.show, (v) => {
  if (v) {
    form.amount = 0; form.bonusAmount = 0; form.bonusValidityDays = 30; form.notes = ''
    void loadBonusSummary()
  }
})

const loadBonusSummary = async () => {
  if (!props.user) return
  try {
    const res = await adminAPI.users.getUserBalanceHistory(props.user.id, 1, 1)
    bonusBalance.value = Math.max(0, res.bonus_balance || 0)
    nearestBonusExpiry.value = res.nearest_bonus_expiry || null
    nearestBonusExpiryAmount.value = Math.max(0, res.nearest_bonus_expiry_amount || 0)
  } catch (error) {
    console.error('Failed to load bonus balance:', error)
  }
}

// 格式化余额：显示完整精度，去除尾部多余的0
const formatBalance = (value: number) => {
  if (value === 0) return '0.00'
  // 最多保留8位小数，去除尾部的0
  const formatted = value.toFixed(8).replace(/\.?0+$/, '')
  // 确保至少有2位小数
  const parts = formatted.split('.')
  if (parts.length === 1) return formatted + '.00'
  if (parts[1].length === 1) return formatted + '0'
  return formatted
}

// 填入全部余额
const fillAllBalance = () => {
  if (props.user) {
    form.amount = props.user.balance
  }
}

const calculateNewBalance = () => {
  if (!props.user) return 0
  const result = props.operation === 'add' ? permanentBalance.value + form.amount : permanentBalance.value - form.amount
  // 避免浮点数精度问题导致的 -0.00 显示
  return Math.abs(result) < 1e-10 ? 0 : result
}
const handleBalanceSubmit = async () => {
  if (!props.user) return
  if (!canSubmit.value) {
    appStore.showError(t('admin.users.amountRequired'))
    return
  }
  if (props.operation === 'add' && form.bonusAmount > 0 && (!form.bonusValidityDays || form.bonusValidityDays < 1 || form.bonusValidityDays > 3650)) {
    appStore.showError(t('admin.users.bonusValidityInvalid'))
    return
  }
  // 退款时验证金额不超过实际余额
  if (props.operation === 'subtract' && form.amount > props.user.balance) {
    appStore.showError(t('admin.users.insufficientBalance'))
    return
  }
  submitting.value = true
  try {
    await adminAPI.users.updateBalance(props.user.id, form.amount, props.operation, form.notes, props.operation === 'add' ? form.bonusAmount : 0, props.operation === 'add' ? form.bonusValidityDays : 0)
    appStore.showSuccess(t('common.success')); emit('success'); emit('close')
  } catch (e: any) {
    console.error('Failed to update balance:', e)
    appStore.showError(e.response?.data?.detail || t('common.error'))
  } finally { submitting.value = false }
}
</script>
