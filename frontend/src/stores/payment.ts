/**
 * Payment Store
 * Manages payment configuration, current order state, and subscription plans
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { paymentAPI } from '@/api/payment'
import type { CheckoutInfoResponse, PaymentConfig, PaymentOrder, SubscriptionPlan, CreateOrderRequest } from '@/types/payment'

export const usePaymentStore = defineStore('payment', () => {
  // ==================== State ====================

  /** Payment configuration from backend */
  const config = ref<PaymentConfig | null>(null)
  /** Currently active order (for payment flow) */
  const currentOrder = ref<PaymentOrder | null>(null)
  /** Available subscription plans */
  const plans = ref<SubscriptionPlan[]>([])
  const bonusBalance = ref(0)
  const permanentBalance = ref(0)
  const nearestBonusExpiry = ref<string | null>(null)
  const nearestBonusExpiryAmount = ref(0)
  const bonusSummaryLoaded = ref(false)

  const configLoading = ref(false)
  const configLoaded = ref(false)
  const bonusSummaryLoading = ref(false)
  let pendingAggregateBalance: number | undefined

  // ==================== Actions ====================

  /** Fetch payment configuration */
  async function fetchConfig(force = false): Promise<PaymentConfig | null> {
    if (configLoaded.value && !force) return config.value
    if (configLoading.value) return config.value

    configLoading.value = true
    try {
      const response = await paymentAPI.getConfig()
      config.value = response.data
      configLoaded.value = true
      return config.value
    } catch (error: unknown) {
      console.error('[payment] Failed to fetch config:', error)
      return null
    } finally {
      configLoading.value = false
    }
  }

  /** Fetch available subscription plans */
  async function fetchPlans(): Promise<SubscriptionPlan[]> {
    try {
      const response = await paymentAPI.getPlans()
      // Backend returns features as newline-separated string; parse to array
      plans.value = (response.data || []).map((p: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
        ...p,
        features: typeof p.features === 'string'
          ? p.features.split('\n').map((f: string) => f.trim()).filter(Boolean)
          : (p.features || []),
      }))
      return plans.value
    } catch (error: unknown) {
      console.error('[payment] Failed to fetch plans:', error)
      return []
    }
  }

  function setBonusSummary(checkout: Pick<CheckoutInfoResponse, 'bonus_balance' | 'nearest_bonus_expiry' | 'nearest_bonus_expiry_amount'>, aggregateBalance?: number) {
    const nextBonusBalance = Math.max(0, Number(checkout.bonus_balance || 0))
    bonusBalance.value = nextBonusBalance
    if (Number.isFinite(aggregateBalance)) {
      permanentBalance.value = Math.max(0, Number(aggregateBalance) - nextBonusBalance)
    }
    nearestBonusExpiry.value = checkout.nearest_bonus_expiry || null
    nearestBonusExpiryAmount.value = Math.max(0, Number(checkout.nearest_bonus_expiry_amount || 0))
    bonusSummaryLoaded.value = true
  }

  async function fetchBonusSummary(aggregateBalance?: number) {
    if (bonusSummaryLoading.value) {
      pendingAggregateBalance = aggregateBalance
      return
    }

    bonusSummaryLoading.value = true
    try {
      const response = await paymentAPI.getCheckoutInfo()
      setBonusSummary(response.data, aggregateBalance)
    } catch (error: unknown) {
      console.error('[payment] Failed to fetch bonus summary:', error)
    } finally {
      bonusSummaryLoading.value = false
      if (pendingAggregateBalance !== undefined) {
        const nextBalance = pendingAggregateBalance
        pendingAggregateBalance = undefined
        void fetchBonusSummary(nextBalance)
      }
    }
  }

  /** Create a new order and set it as current */
  async function createOrder(params: CreateOrderRequest) {
    const response = await paymentAPI.createOrder(params)
    return response.data
  }

  /** Poll order status by ID (read-only, no upstream check) */
  async function pollOrderStatus(orderId: number): Promise<PaymentOrder | null> {
    try {
      const response = await paymentAPI.getOrder(orderId)
      const order = response.data
      if (currentOrder.value?.id === orderId) {
        currentOrder.value = order
      }
      return order
    } catch (error: unknown) {
      console.error('[payment] Failed to poll order status:', error)
      return null
    }
  }

  /** Clear current order state */
  function clearCurrentOrder() {
    currentOrder.value = null
  }

  return {
    config,
    currentOrder,
    plans,
    bonusBalance,
    permanentBalance,
    nearestBonusExpiry,
    nearestBonusExpiryAmount,
    bonusSummaryLoaded,
    configLoading,
    configLoaded,
    bonusSummaryLoading,
    fetchConfig,
    fetchPlans,
    setBonusSummary,
    fetchBonusSummary,
    createOrder,
    pollOrderStatus,
    clearCurrentOrder
  }
})
