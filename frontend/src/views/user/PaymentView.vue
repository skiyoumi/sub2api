<template>
  <AppLayout>
    <div class="mx-auto space-y-6" :class="activeTab === 'recharge' && paymentPhase === 'select' ? 'max-w-7xl' : 'max-w-4xl'">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      </div>
      <template v-else>
        <!-- Tab Switcher (hide during payment and subscription confirm) -->
        <div v-if="tabs.length > 1 && paymentPhase === 'select' && !selectedPlan" class="flex space-x-1 rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
          <button v-for="tab in tabs" :key="tab.key"
            class="flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-all"
            :class="activeTab === tab.key ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
            @click="activeTab = tab.key">{{ tab.label }}</button>
        </div>
        <!-- Payment in progress (shared by recharge and subscription) -->
        <template v-if="paymentPhase === 'paying'">
          <PaymentStatusPanel
            :order-id="paymentState.orderId"
            :qr-code="paymentState.qrCode"
            :expires-at="paymentState.expiresAt"
            :payment-type="paymentState.paymentType"
            :pay-url="paymentState.payUrl"
            :order-type="paymentState.orderType"
            :currency="paymentState.currency || selectedCurrency"
            @done="onPaymentDone"
            @success="onPaymentSuccess"
            @settled="onPaymentSettled"
          />
        </template>
        <!-- Tab content (select phase) -->
        <template v-else>
          <!-- Top-up Tab -->
          <template v-if="activeTab === 'recharge'">
            <div v-if="enabledMethods.length === 0" class="card py-16 text-center">
              <p class="text-gray-500 dark:text-gray-400">{{ t('payment.notAvailable') }}</p>
            </div>
            <div v-else class="recharge-checkout-shell">
              <header class="recharge-cashier-header">
                <div class="flex min-w-0 items-center gap-3">
                  <span class="recharge-cashier-icon"><Icon name="creditCard" size="md" /></span>
                  <div class="min-w-0">
                    <h1>{{ t('payment.rechargePackages.cashierTitle') }}</h1>
                    <p>{{ t('payment.rechargePackages.cashierSubtitle') }}</p>
                  </div>
                </div>
                <span class="recharge-security-badge">
                  <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
                  {{ t('payment.rechargePackages.encrypted') }}
                </span>
              </header>

              <div class="recharge-workspace grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
                <div class="space-y-5">
                <section class="recharge-panel">
                  <div class="recharge-step-header">
                    <span class="recharge-step-number">1</span>
                    <div>
                      <h2>{{ t('payment.rechargePackages.chooseAmount') }}</h2>
                      <p>{{ t('payment.rechargePackages.chooseAmountHint') }}</p>
                    </div>
                    <span v-if="globalMaxAmount > 0" class="ml-auto hidden rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-950/50 dark:text-blue-300 sm:block">
                      {{ t('payment.rechargePackages.maximum', { amount: formatSelectedPaymentAmount(globalMaxAmount) }) }}
                    </span>
                  </div>

                  <div v-if="checkout.recharge_packages_enabled && rechargePackages.length" class="mt-6">
                    <div class="mb-3 flex items-center justify-between gap-4">
                      <p class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('payment.rechargePackages.title') }}</p>
                      <button v-if="checkout.allow_custom_amount" class="text-xs font-medium text-primary-600 hover:underline dark:text-primary-400" @click="clearRechargePackage">
                        {{ t('payment.rechargePackages.custom') }}
                      </button>
                    </div>
                    <div v-if="hasBonusPackages" class="recharge-bonus-notice">
                      <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
                      <div>
                        <p class="font-semibold">{{ t('payment.rechargePackages.bonusPolicyTitle') }}</p>
                        <p class="mt-0.5">{{ t('payment.rechargePackages.bonusPolicyDescription') }}</p>
                      </div>
                    </div>
                    <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4" role="radiogroup" :aria-label="t('payment.rechargePackages.title')">
                      <button v-for="pkg in rechargePackages" :key="pkg.id" type="button" role="radio" :aria-checked="selectedRechargePackageId === pkg.id"
                        class="recharge-package-card group"
                        :class="selectedRechargePackageId === pkg.id ? 'is-selected' : ''"
                        @click="selectRechargePackage(pkg)">
                        <span v-if="pkg.recommended" class="recharge-recommended">{{ t('payment.rechargePackages.recommended') }}</span>
                        <span class="flex items-baseline gap-1 text-gray-950 dark:text-white">
                          <span class="text-sm font-semibold text-gray-500 dark:text-gray-400">{{ selectedCurrencySymbol }}</span>
                          <span class="text-3xl font-bold leading-none">{{ formatPackageFaceAmount(pkg.amount) }}</span>
                        </span>
                        <span class="mt-2 block text-xs text-gray-400 dark:text-gray-500">{{ t('payment.rechargePackages.officialCredit') }}</span>
                        <span v-if="pkg.bonus_amount > 0" class="recharge-bonus-pill">
                          <Icon name="gift" size="xs" />
                          {{ t('payment.rechargePackages.bonus', { amount: `${selectedCurrencySymbol}${formatPackageFaceAmount(pkg.bonus_amount)}`, days: pkg.bonus_validity_days }) }}
                        </span>
                        <span v-else class="mt-4 inline-flex rounded-full bg-gray-100 px-2 py-1 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                          {{ t('payment.rechargePackages.instantCredit') }}
                        </span>
                        <span v-if="selectedRechargePackageId === pkg.id" class="recharge-selected-check">✓</span>
                      </button>
                    </div>
                  </div>

                  <div v-if="!checkout.recharge_packages_enabled || !selectedRechargePackageId" class="mt-5">
                    <AmountInput v-model="amount" :amounts="[10, 20, 50, 100, 200, 500, 1000, 2000, 5000]" :min="globalMinAmount" :max="globalMaxAmount" />
                  </div>
                  <p v-if="amountError" class="mt-3 text-xs text-amber-600 dark:text-amber-300">{{ amountError }}</p>
                </section>

                <section class="recharge-panel">
                  <div class="recharge-step-header">
                    <span class="recharge-step-number">2</span>
                    <div>
                      <h2>{{ t('payment.rechargePackages.chooseMethod') }}</h2>
                      <p>{{ t('payment.rechargePackages.chooseMethodHint') }}</p>
                    </div>
                  </div>
                  <div class="mt-6">
                    <PaymentMethodSelector :methods="methodOptions" :selected="selectedMethod" @select="selectedMethod = $event" />
                  </div>
                </section>
                </div>

                <aside class="space-y-5 lg:sticky lg:top-20">
                <section class="recharge-summary-panel">
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('payment.actualPay') }}</p>
                  <p class="mt-2 text-5xl font-bold text-primary-600 dark:text-primary-400">{{ formatSelectedPaymentAmount(totalAmount) }}</p>
                  <div class="mt-7 space-y-3 border-t border-gray-100 pt-5 text-sm dark:border-dark-700">
                    <div class="flex justify-between gap-4">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('payment.paymentAmount') }}</span>
                      <span class="font-medium text-gray-900 dark:text-white">{{ formatSelectedPaymentAmount(validAmount) }}</span>
                    </div>
                    <div v-if="feeRate > 0" class="flex justify-between gap-4">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                      <span class="font-medium text-gray-900 dark:text-white">{{ formatSelectedPaymentAmount(feeAmount) }}</span>
                    </div>
                    <div class="flex justify-between gap-4">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('payment.creditedBalance') }}</span>
                      <span class="font-semibold text-emerald-600 dark:text-emerald-400">${{ creditedAmount.toFixed(2) }}</span>
                    </div>
                  </div>
                  <button :class="['btn mt-6 hidden w-full py-3 text-base font-semibold lg:flex', paymentButtonClass]" :disabled="!canSubmit || submitting" @click="handleSubmitRecharge">
                    <span v-if="submitting" class="flex items-center justify-center gap-2">
                      <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                      {{ t('common.processing') }}
                    </span>
                    <span v-else>{{ t('payment.rechargePackages.confirmPay', { amount: formatSelectedPaymentAmount(totalAmount) }) }}</span>
                  </button>
                </section>

                <section class="recharge-account-panel">
                  <div class="mb-5 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                    <span class="flex h-9 w-9 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"><Icon name="user" size="sm" /></span>
                    <span>{{ t('payment.rechargeAccount') }}</span>
                  </div>
                  <div class="grid grid-cols-2 gap-3">
                    <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.currentBalance') }}</p>
                      <p class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ permanentBalance.toFixed(2) }}</p>
                    </div>
                    <div class="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30">
                      <p class="text-xs text-primary-600 dark:text-primary-400">{{ t('payment.rechargePackages.balanceAfter') }}</p>
                      <p class="mt-1 text-lg font-bold text-primary-700 dark:text-primary-300">{{ estimatedBalance.toFixed(2) }}</p>
                    </div>
                  </div>
                  <div v-if="currentBonusBalance > 0" class="mt-3 space-y-1 text-xs text-red-600 dark:text-red-400">
                    <p>{{ t('payment.rechargePackages.currentBonus', { amount: currentBonusBalance.toFixed(2) }) }}</p>
                    <p v-if="checkout.nearest_bonus_expiry && nearestBonusExpiryAmount > 0">
                      {{ t('payment.rechargePackages.bonusExpiry', { amount: nearestBonusExpiryAmount.toFixed(2), date: formatDateTimeToMinute(checkout.nearest_bonus_expiry) }) }}
                    </p>
                  </div>
                </section>

                <section v-if="checkout.help_text || checkout.help_image_url || supportContact" class="recharge-help-panel">
                  <div class="flex gap-3">
                    <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0 text-amber-600" />
                    <div class="min-w-0">
                      <p class="font-semibold text-amber-900 dark:text-amber-200">{{ t('payment.rechargePackages.helpTitle') }}</p>
                      <p v-if="checkout.help_text" class="mt-2 text-sm leading-6 text-amber-800/80 dark:text-amber-300/80">{{ checkout.help_text }}</p>
                    </div>
                  </div>
                  <img v-if="checkout.help_image_url" :src="checkout.help_image_url" alt="" class="mx-auto mt-4 max-h-36 max-w-full cursor-pointer rounded-lg object-contain" @click="previewImage = checkout.help_image_url" />
                  <button v-if="supportContact" type="button" class="recharge-contact-button" @click="copySupportContact">
                    <Icon name="copy" size="sm" />
                    {{ t('payment.rechargePackages.contactService') }}
                  </button>
                </section>
                </aside>
              </div>
            </div>

            <footer v-if="enabledMethods.length > 0" class="recharge-risk-footer">
              <Icon name="shield" size="sm" class="shrink-0 text-gray-400 dark:text-gray-500" />
              <p>{{ t('payment.rechargePackages.riskNotice') }}</p>
            </footer>

            <div v-if="enabledMethods.length > 0" class="fixed inset-x-0 bottom-0 z-30 border-t border-gray-200 bg-white/95 p-3 shadow-[0_-8px_30px_rgba(15,23,42,0.08)] backdrop-blur dark:border-dark-700 dark:bg-dark-900/95 lg:hidden">
              <div class="mx-auto flex max-w-7xl items-center gap-3">
                <div class="min-w-0 flex-1">
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.actualPay') }}</p>
                  <p class="truncate text-xl font-bold text-primary-600 dark:text-primary-400">{{ formatSelectedPaymentAmount(totalAmount) }}</p>
                </div>
                <button :class="['btn min-w-[160px] px-5 py-3 font-semibold', paymentButtonClass]" :disabled="!canSubmit || submitting" @click="handleSubmitRecharge">
                  {{ submitting ? t('common.processing') : t('payment.rechargePackages.confirmPayShort') }}
                </button>
              </div>
            </div>
          </template>
          <!-- Subscribe Tab -->
          <template v-else-if="activeTab === 'subscription'">
            <!-- Subscription confirm (inline, replaces plan list) -->
            <template v-if="selectedPlan">
              <div class="card p-5">
                <!-- Header: platform badge + plan name -->
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <span :class="['rounded-md border px-2 py-0.5 text-xs font-medium', planBadgeClass]">
                    {{ platformLabel(selectedPlan.group_platform || '') }}
                  </span>
                  <h3 class="text-lg font-bold text-gray-900 dark:text-white">{{ selectedPlan.name }}</h3>
                </div>
                <!-- Price -->
                <div class="flex items-baseline gap-2">
                  <span v-if="selectedPlan.original_price" class="text-sm text-gray-400 line-through dark:text-gray-500">
                    {{ formatSelectedSubscriptionPaymentAmount(selectedPlan.original_price) }}
                  </span>
                  <span :class="['text-3xl font-bold', planTextClass]">{{ formatSelectedSubscriptionPaymentAmount(selectedPlan.price) }}</span>
                  <span class="text-sm text-gray-500 dark:text-gray-400">/ {{ planValiditySuffix }}</span>
                </div>
                <!-- Description -->
                <p v-if="selectedPlan.description" class="mt-2 text-sm leading-relaxed text-gray-500 dark:text-gray-400">
                  {{ selectedPlan.description }}
                </p>
                <!-- Rate + Limits grid -->
                <div class="mt-3 grid grid-cols-2 gap-3">
                  <div>
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.rate') }}</span>
                    <div class="flex items-baseline">
                      <span :class="['text-lg font-bold', planTextClass]">×{{ selectedPlan.rate_multiplier ?? 1 }}</span>
                    </div>
                  </div>
                  <div v-if="planHasPeakRate(selectedPlan)">
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.peakRate') }}</span>
                    <div class="text-sm font-semibold text-amber-700 dark:text-amber-300">
                      {{ planPeakRateLabel(selectedPlan) }}
                    </div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd != null">
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.dailyLimit') }}</span>
                    <div class="text-lg font-semibold text-gray-800 dark:text-gray-200">${{ selectedPlan.daily_limit_usd }}</div>
                  </div>
                  <div v-if="selectedPlan.weekly_limit_usd != null">
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.weeklyLimit') }}</span>
                    <div class="text-lg font-semibold text-gray-800 dark:text-gray-200">${{ selectedPlan.weekly_limit_usd }}</div>
                  </div>
                  <div v-if="selectedPlan.monthly_limit_usd != null">
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.monthlyLimit') }}</span>
                    <div class="text-lg font-semibold text-gray-800 dark:text-gray-200">${{ selectedPlan.monthly_limit_usd }}</div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd == null && selectedPlan.weekly_limit_usd == null && selectedPlan.monthly_limit_usd == null">
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.quota') }}</span>
                    <div class="text-lg font-semibold text-gray-800 dark:text-gray-200">{{ t('payment.planCard.unlimited') }}</div>
                  </div>
                </div>
              </div>
              <div v-if="enabledMethods.length >= 1" class="card p-6">
                <PaymentMethodSelector
                  :methods="subMethodOptions"
                  :selected="selectedMethod"
                  @select="selectedMethod = $event"
                />
              </div>
              <div v-if="feeRate > 0 && selectedPlan.price > 0" class="card p-6">
                <div class="space-y-2 text-sm">
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('payment.amountLabel') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ formatSelectedPaymentAmount(subPaymentAmount) }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                    <span class="text-gray-900 dark:text-white">{{ formatSelectedPaymentAmount(subFeeAmount) }}</span>
                  </div>
                  <div class="flex justify-between border-t border-gray-200 pt-2 dark:border-dark-600">
                    <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.actualPay') }}</span>
                    <span class="text-lg font-bold text-primary-600 dark:text-primary-400">{{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
                  </div>
                </div>
              </div>
              <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmitSubscription || submitting" @click="confirmSubscribe">
                <span v-if="submitting" class="flex items-center justify-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  {{ t('common.processing') }}
                </span>
                <span v-else>{{ t('payment.createOrder') }} {{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
              </button>
              <button class="btn btn-secondary w-full" @click="selectedPlan = null">{{ t('common.cancel') }}</button>
            </template>
            <!-- Plan list -->
            <template v-else>
              <div v-if="checkout.plans.length === 0" class="card py-16 text-center">
                <Icon name="gift" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
                <p class="text-gray-500 dark:text-gray-400">{{ t('payment.noPlans') }}</p>
              </div>
              <div v-else :class="planGridClass">
                <SubscriptionPlanCard v-for="plan in checkout.plans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlan" />
              </div>
              <!-- Active subscriptions (compact, below plan list) -->
              <div v-if="activeSubscriptions.length > 0">
                <p class="mb-2 text-xs font-medium text-gray-400 dark:text-gray-500">{{ t('payment.activeSubscription') }}</p>
                <div class="space-y-2">
                  <div v-for="sub in activeSubscriptions" :key="sub.id"
                    class="flex items-center gap-3 rounded-xl border border-gray-100 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
                    <div :class="['h-6 w-1 shrink-0 rounded-full', platformAccentBarClass(sub.group?.platform || '')]" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="truncate text-xs font-semibold text-gray-900 dark:text-white">{{ sub.group?.name || t('payment.groupFallback', { id: sub.group_id }) }}</span>
                        <span :class="['shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-medium', platformBadgeLightClass(sub.group?.platform || '')]">{{ platformLabel(sub.group?.platform || '') }}</span>
                      </div>
                      <div class="flex flex-wrap gap-x-3 text-[11px] text-gray-400 dark:text-gray-500">
                        <span>{{ t('payment.planCard.rate') }}: ×{{ sub.group?.rate_multiplier ?? 1 }}</span>
                        <span v-if="subscriptionHasPeakRate(sub)">{{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(sub) }}</span>
                        <span v-if="sub.group?.daily_limit_usd == null && sub.group?.weekly_limit_usd == null && sub.group?.monthly_limit_usd == null">{{ t('payment.planCard.quota') }}: {{ t('payment.planCard.unlimited') }}</span>
                        <span v-if="sub.expires_at">{{ t('userSubscriptions.daysRemaining', { days: getDaysRemaining(sub.expires_at) }) }}</span>
                        <span v-else>{{ t('userSubscriptions.noExpiration') }}</span>
                      </div>
                    </div>
                    <span class="badge badge-success shrink-0 text-[10px]">{{ t('userSubscriptions.status.active') }}</span>
                  </div>
                </div>
              </div>
            </template>
          </template>
        </template>
        <div v-if="(checkout.help_text || checkout.help_image_url) && paymentPhase === 'select' && !selectedPlan && activeTab !== 'recharge'" class="card p-4">
          <div class="flex flex-col items-center gap-3">
            <img v-if="checkout.help_image_url" :src="checkout.help_image_url" alt=""
              class="h-40 max-w-full cursor-pointer rounded-lg object-contain transition-opacity hover:opacity-80"
              @click="previewImage = checkout.help_image_url" />
            <p v-if="checkout.help_text" class="text-center text-sm text-gray-500 dark:text-gray-400">{{ checkout.help_text }}</p>
          </div>
        </div>
      </template>
    </div>
    <!-- Renewal Plan Selection Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showRenewalModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4" @click.self="closeRenewalModal">
          <div class="relative w-full max-w-lg rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl dark:border-dark-700 dark:bg-dark-900">
            <!-- Close button -->
            <button class="absolute right-4 top-4 rounded-lg p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-200" @click="closeRenewalModal">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
            <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('payment.selectPlan') }}</h3>
            <div class="space-y-4">
              <SubscriptionPlanCard v-for="plan in renewalPlans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlanFromModal" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
    <!-- Image Preview Overlay -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="previewImage" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 backdrop-blur-sm" @click="previewImage = ''">
          <img :src="previewImage" alt="" class="max-h-[85vh] max-w-[90vw] rounded-xl object-contain shadow-2xl" />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel, type PeakRateFields } from '@/utils/peak-rate'
import type { SubscriptionPlan, CheckoutInfoResponse, CreateOrderResult, OrderType, RechargePackage } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import { METHOD_ORDER, getPaymentPopupFeatures, isBuiltInAlipayMethod, isBuiltInWxpayMethod } from '@/components/payment/providerConfig'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  buildCreateOrderPayload,
  clearPaymentRecoverySnapshot,
  decidePaymentLaunch,
  getVisibleMethods,
  normalizeVisibleMethod,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
  writePaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { platformAccentBarClass, platformBadgeLightClass, platformBadgeClass, platformTextClass, platformLabel } from '@/utils/platformColors'
import SubscriptionPlanCard from '@/components/payment/SubscriptionPlanCard.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { DEFAULT_PAYMENT_CURRENCY, formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import { planValiditySuffix as validitySuffixOf } from '@/components/payment/validity'
import { formatDateTimeToMinute } from '@/utils/format'
import type { PaymentMethodOption } from '@/components/payment/PaymentMethodSelector.vue'
import { buildPaymentErrorToastMessage, describePaymentScenarioError } from './paymentUx'
import { hasWechatResumeQuery, parseWechatResumeRoute, stripWechatResumeQuery } from './paymentWechatResume'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const subscriptionStore = useSubscriptionStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const user = computed(() => authStore.user)
const supportContact = computed(() => (appStore.contactInfo || '').trim())
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)

function copySupportContact() {
  if (!supportContact.value) return
  void copyToClipboard(supportContact.value, t('common.copiedToClipboard'))
}

function getDaysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

function subscriptionHasPeakRate(sub: { group?: PeakRateFields | null }): boolean {
  return hasPeakRate(sub.group)
}

function subscriptionPeakRateLabel(sub: { group?: PeakRateFields | null }): string {
  return formatPeakRateWindow(sub.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

const loading = ref(true)
const submitting = ref(false)
const errorMessage = ref('')
const errorHintMessage = ref('')
const activeTab = ref<'recharge' | 'subscription'>('recharge')
const amount = ref<number | null>(null)
const selectedRechargePackageId = ref('')
const selectedMethod = ref('')
const selectedPlan = ref<SubscriptionPlan | null>(null)
const previewImage = ref('')

const paymentPhase = ref<'select' | 'paying'>('select')

interface CreateOrderOptions {
  openid?: string
  wechatResumeToken?: string
  paymentType?: string
  isResume?: boolean
  mobileQrFallbackAttempted?: boolean
  rechargePackageId?: string
}

interface WeixinJSBridgeLike {
  invoke(
    action: string,
    payload: Record<string, unknown>,
    callback: (result: Record<string, unknown>) => void,
  ): void
}

function emptyPaymentState(): PaymentRecoverySnapshot {
  return {
    orderId: 0,
    amount: 0,
    qrCode: '',
    expiresAt: '',
    paymentType: '',
    payUrl: '',
    outTradeNo: '',
    clientSecret: '',
    intentId: '',
    currency: '',
    countryCode: '',
    paymentEnv: '',
    payAmount: 0,
    orderType: '',
    paymentMode: '',
    resumeToken: '',
    createdAt: 0,
  }
}

function getWeixinJSBridge(): WeixinJSBridgeLike | undefined {
  return (window as Window & { WeixinJSBridge?: WeixinJSBridgeLike }).WeixinJSBridge
}

function waitForWeixinJSBridge(timeoutMs = 4000): Promise<WeixinJSBridgeLike | null> {
  const existing = getWeixinJSBridge()
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    let settled = false
    const finish = (bridge: WeixinJSBridgeLike | null) => {
      if (settled) return
      settled = true
      document.removeEventListener('WeixinJSBridgeReady', handleReady)
      document.removeEventListener('onWeixinJSBridgeReady', handleReady)
      window.clearTimeout(timer)
      resolve(bridge)
    }
    const handleReady = () => finish(getWeixinJSBridge() ?? null)
    const timer = window.setTimeout(() => finish(getWeixinJSBridge() ?? null), timeoutMs)
    document.addEventListener('WeixinJSBridgeReady', handleReady, false)
    document.addEventListener('onWeixinJSBridgeReady', handleReady, false)
  })
}

async function invokeWechatJsapiPayment(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  const bridge = await waitForWeixinJSBridge()
  if (!bridge) {
    throw new Error('WECHAT_JSAPI_UNAVAILABLE')
  }
  return new Promise((resolve) => {
    bridge.invoke('getBrandWCPayRequest', payload, (result) => resolve(result || {}))
  })
}

const paymentState = ref<PaymentRecoverySnapshot>(emptyPaymentState())

function persistRecoverySnapshot(snapshot: PaymentRecoverySnapshot) {
  if (typeof window === 'undefined' || !snapshot.orderId) return
  writePaymentRecoverySnapshot(window.localStorage, snapshot, PAYMENT_RECOVERY_STORAGE_KEY)
}

function removeRecoverySnapshot() {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function resetPayment() {
  paymentPhase.value = 'select'
  paymentState.value = emptyPaymentState()
  removeRecoverySnapshot()
}

async function redirectToPaymentResult(state: PaymentRecoverySnapshot): Promise<void> {
  const query: Record<string, string | undefined> = {}
  if (state.orderId > 0) {
    query.order_id = String(state.orderId)
  }
  if (state.outTradeNo) {
    query.out_trade_no = state.outTradeNo
  }
  if (state.resumeToken) {
    query.resume_token = state.resumeToken
  }
  await router.push({
    path: '/payment/result',
    query,
  })
}

function buildWechatOAuthAuthorizeUrl(
  authorizeUrl: string,
  context: { paymentType: string; orderType: OrderType; planId?: number; orderAmount: number },
): string {
  const normalizedUrl = authorizeUrl.trim()
  if (!normalizedUrl || typeof window === 'undefined') {
    return normalizedUrl
  }

  try {
    const targetUrl = new URL(normalizedUrl, window.location.origin)
    const redirectPath = targetUrl.searchParams.get('redirect') || '/purchase'
    const redirectUrl = new URL(redirectPath, window.location.origin)
    const paymentType = normalizeVisibleMethod(context.paymentType) || context.paymentType.trim() || 'wxpay'

    redirectUrl.searchParams.set('payment_type', paymentType)
    redirectUrl.searchParams.set('order_type', context.orderType)

    if (context.planId) {
      redirectUrl.searchParams.set('plan_id', String(context.planId))
    } else {
      redirectUrl.searchParams.delete('plan_id')
    }

    if (context.orderAmount > 0) {
      redirectUrl.searchParams.set('amount', String(context.orderAmount))
    } else {
      redirectUrl.searchParams.delete('amount')
    }

    targetUrl.searchParams.set('redirect', `${redirectUrl.pathname}${redirectUrl.search}`)
    return targetUrl.toString()
  } catch {
    return normalizedUrl
  }
}

function onPaymentDone() {
  const wasSubscription = paymentState.value.orderType === 'subscription'
  resetPayment()
  selectedPlan.value = null
  if (wasSubscription) {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSuccess() {
  removeRecoverySnapshot()
  authStore.refreshUser()
  if (paymentState.value.orderType === 'subscription') {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSettled() {
  removeRecoverySnapshot()
}

// All checkout data from single API call
const checkout = ref<CheckoutInfoResponse>({
  methods: {}, global_min: 0, global_max: 0,
  plans: [], balance_disabled: false, balance_recharge_multiplier: 1, subscription_usd_to_cny_rate: 0, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '',
})

const rechargePackages = computed(() => checkout.value.recharge_packages || [])
const selectedRechargePackage = computed(() => rechargePackages.value.find(pkg => pkg.id === selectedRechargePackageId.value) || null)
const hasBonusPackages = computed(() => rechargePackages.value.some(pkg => pkg.bonus_amount > 0))

function selectRechargePackage(pkg: RechargePackage) {
  selectedRechargePackageId.value = pkg.id
  amount.value = pkg.amount
}

function clearRechargePackage() {
  selectedRechargePackageId.value = ''
}

const tabs = computed(() => {
  const result: { key: 'recharge' | 'subscription'; label: string }[] = []
  if (!checkout.value.balance_disabled) result.push({ key: 'recharge', label: t('payment.tabTopUp') })
  result.push({ key: 'subscription', label: t('payment.tabSubscribe') })
  return result
})

const visibleMethods = computed(() => getVisibleMethods(checkout.value.methods))
const enabledMethods = computed(() => Object.keys(visibleMethods.value))
const validAmount = computed(() => amount.value ?? 0)
const balanceRechargeMultiplier = computed(() => {
  const multiplier = checkout.value.balance_recharge_multiplier
  return Number.isFinite(multiplier) && multiplier > 0 ? multiplier : 1
})
// 订阅 CNY 换算汇率（1 USD = X CNY）。0 = 未配置，订阅保持 price 直付（与后端 opt-in 条件严格镜像）。
const subscriptionUsdToCnyRate = computed(() => {
  const rate = checkout.value.subscription_usd_to_cny_rate
  return Number.isFinite(rate) && rate > 0 ? rate : 0
})
const currentBonusBalance = computed(() => Math.max(0, checkout.value.bonus_balance || 0))
const nearestBonusExpiryAmount = computed(() => Math.max(0, checkout.value.nearest_bonus_expiry_amount || 0))
const permanentBalance = computed(() => paymentStore.bonusSummaryLoaded
  ? Math.round(paymentStore.permanentBalance * 100) / 100
  : Math.max(0, Math.round((user.value?.balance || 0) * 100) / 100))
const permanentCreditedAmount = computed(() => Math.round((validAmount.value * balanceRechargeMultiplier.value) * 100) / 100)
const creditedAmount = computed(() => Math.round((permanentCreditedAmount.value + (selectedRechargePackage.value?.bonus_amount || 0)) * 100) / 100)
const estimatedBalance = computed(() => Math.round((permanentBalance.value + permanentCreditedAmount.value) * 100) / 100)

// Adaptive grid: center single card, 2-col for 2 plans, 3-col for 3+
const planGridClass = computed(() => {
  const n = checkout.value.plans.length
  if (n <= 2) return 'grid grid-cols-1 gap-5 sm:grid-cols-2'
  return 'grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3'
})

// Check if an amount fits a method's [min, max]. 0 = no limit.
function amountFitsMethod(amt: number, methodType: string): boolean {
  if (amt <= 0) return true
  const ml = visibleMethods.value[methodType]
  if (!ml) return false
  if (ml.single_min > 0 && amt < ml.single_min) return false
  if (ml.single_max > 0 && amt > ml.single_max) return false
  return true
}

// Visible methods decide the amount range shown to users.
const globalMinAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_min <= 0)) return 0
  return Math.min(...limits.map(limit => limit.single_min))
})
const globalMaxAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_max <= 0)) return 0
  return Math.max(...limits.map(limit => limit.single_max))
})

// Selected method's limits (for validation and error messages)
const selectedLimit = computed(() => visibleMethods.value[selectedMethod.value])
const selectedCurrency = computed(() => normalizePaymentCurrency(selectedLimit.value?.currency))
const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})
const selectedCurrencySymbol = computed(() => {
  try {
    return new Intl.NumberFormat(localeCode.value, {
      style: 'currency',
      currency: selectedCurrency.value,
      currencyDisplay: 'narrowSymbol',
    }).formatToParts(0).find(part => part.type === 'currency')?.value || selectedCurrency.value
  } catch {
    return selectedCurrency.value
  }
})
function currencyFractionDigits(currency: string): number {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency,
    }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function roundPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.round(value * factor) / factor
}

function ceilPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.ceil(value * factor) / factor
}

function subscriptionPaymentAmountForCurrency(value: number, currency: string): number {
  const rate = subscriptionUsdToCnyRate.value
  if (rate <= 0 || currency !== DEFAULT_PAYMENT_CURRENCY) return roundPaymentAmount(value, currency)
  return roundPaymentAmount(value * rate, currency)
}

function formatSelectedPaymentAmount(value: number): string {
  return formatPaymentAmount(value, selectedCurrency.value, localeCode.value)
}

function formatPackageFaceAmount(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
}

function formatSelectedSubscriptionPaymentAmount(value: number): string {
  return formatSelectedPaymentAmount(subscriptionPaymentAmountForCurrency(value, selectedCurrency.value))
}

const methodOptions = computed<PaymentMethodOption[]>(() =>
  enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(validAmount.value, type),
    }
  })
)

const feeRate = computed(() => checkout.value?.recharge_fee_rate ?? 0)
const feeAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.ceil(((validAmount.value * feeRate.value) / 100) * 100) / 100
    : 0
)
const totalAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.round((validAmount.value + feeAmount.value) * 100) / 100
    : validAmount.value
)

const amountError = computed(() => {
  if (validAmount.value <= 0) return ''
  // No method can handle this amount
  if (!enabledMethods.value.some((m) => amountFitsMethod(validAmount.value, m))) {
    return t('payment.amountNoMethod')
  }
  // Selected method can't handle this amount (but others can)
  const ml = selectedLimit.value
  if (ml) {
    if (ml.single_min > 0 && validAmount.value < ml.single_min) return t('payment.amountTooLow', { min: formatSelectedPaymentAmount(ml.single_min) })
    if (ml.single_max > 0 && validAmount.value > ml.single_max) return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(ml.single_max) })
  }
  return ''
})

const canSubmit = computed(() =>
  validAmount.value > 0
    && amountFitsMethod(validAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

const subPaymentAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  return subscriptionPaymentAmountForCurrency(price, selectedCurrency.value)
})

const subFeeAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return 0
  return ceilPaymentAmount((subPaymentAmount.value * feeRate.value) / 100, selectedCurrency.value)
})

const subTotalAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return subPaymentAmount.value
  return roundPaymentAmount(subPaymentAmount.value + subFeeAmount.value, selectedCurrency.value)
})

function subscriptionTotalAmountForCurrency(value: number, currency: string): number {
  const paymentAmount = subscriptionPaymentAmountForCurrency(value, currency)
  if (feeRate.value <= 0 || paymentAmount <= 0) return paymentAmount
  const fee = ceilPaymentAmount((paymentAmount * feeRate.value) / 100, currency)
  return roundPaymentAmount(paymentAmount + fee, currency)
}

// Subscription-specific: method options based on gateway pay amount
const subMethodOptions = computed<PaymentMethodOption[]>(() => {
  const price = selectedPlan.value?.price ?? 0
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    const currency = normalizePaymentCurrency(ml?.currency)
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(subscriptionTotalAmountForCurrency(price, currency), type),
    }
  })
})

const canSubmitSubscription = computed(() =>
  selectedPlan.value !== null
    && amountFitsMethod(subTotalAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Auto-switch to first available method when current selection can't handle the amount
watch(() => [validAmount.value, selectedMethod.value] as const, ([amt, method]) => {
  if (amt <= 0 || amountFitsMethod(amt, method)) return
  const available = enabledMethods.value.find((m) => amountFitsMethod(amt, m))
  if (available) selectedMethod.value = available
})

watch(amount, (value) => {
  const pkg = selectedRechargePackage.value
  if (pkg && value !== pkg.amount) {
    selectedRechargePackageId.value = ''
  }
})

// Payment button class: follows selected payment method color
const paymentButtonClass = computed(() => {
  const m = selectedMethod.value
  if (!m) return 'btn-primary'
  if (isBuiltInAlipayMethod(m)) return 'btn-alipay'
  if (isBuiltInWxpayMethod(m)) return 'btn-wxpay'
  if (m === 'stripe') return 'btn-stripe'
  if (m === 'airwallex') return 'btn-airwallex'
  return 'btn-primary'
})

// Subscription confirm: platform accent colors (clean card, no gradient)
const planBadgeClass = computed(() => platformBadgeClass(selectedPlan.value?.group_platform || ''))
const planTextClass = computed(() => platformTextClass(selectedPlan.value?.group_platform || ''))

// Renewal modal state
const showRenewalModal = ref(false)
const renewGroupId = ref<number | null>(null)
const renewalPlans = computed(() => {
  if (renewGroupId.value == null) return []
  return checkout.value.plans.filter(p => p.group_id === renewGroupId.value)
})

const planValiditySuffix = computed(() => {
  if (!selectedPlan.value) return ''
  return validitySuffixOf(selectedPlan.value, t)
})

function planHasPeakRate(plan: SubscriptionPlan): boolean {
  return hasPeakRate(plan)
}

function planPeakRateLabel(plan: SubscriptionPlan): string {
  return formatPeakRateWindow(plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function selectPlan(plan: SubscriptionPlan) {
  selectedPlan.value = plan
  errorMessage.value = ''
}

function selectPlanFromModal(plan: SubscriptionPlan) {
  showRenewalModal.value = false
  renewGroupId.value = null
  selectedPlan.value = plan
  errorMessage.value = ''
}

function closeRenewalModal() {
  showRenewalModal.value = false
  renewGroupId.value = null
}

async function handleSubmitRecharge() {
  if (!canSubmit.value || submitting.value) return
  await createOrder(validAmount.value, 'balance', undefined, { rechargePackageId: selectedRechargePackageId.value || undefined })
}

async function confirmSubscribe() {
  if (!selectedPlan.value || submitting.value) return
  await createOrder(selectedPlan.value.price, 'subscription', selectedPlan.value.id)
}

async function createOrder(orderAmount: number, orderType: OrderType, planId?: number, options: CreateOrderOptions = {}) {
  submitting.value = true
  errorMessage.value = ''
  errorHintMessage.value = ''
  const requestType = normalizeVisibleMethod(options.paymentType || selectedMethod.value) || options.paymentType || selectedMethod.value
  try {
    const payload = buildCreateOrderPayload({
      amount: orderAmount,
      paymentType: requestType,
      orderType,
      planId,
      rechargePackageId: orderType === 'balance' ? options.rechargePackageId : undefined,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && normalizeVisibleMethod(requestType) === 'alipay'),
    })
    if (options.openid) {
      payload.openid = options.openid
    }
    if (options.wechatResumeToken) {
      payload.wechat_resume_token = options.wechatResumeToken
    }

    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const openWindow = (url: string) => {
      const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
      if (!win || win.closed) {
        window.location.href = url
      }
    }
    const visibleMethod = normalizeVisibleMethod(requestType) || requestType
    // When user clicks the dedicated Stripe button, leave method blank so the
    // landing page renders Stripe's full Payment Element (card/link/alipay/wxpay).
    const stripeMethod = visibleMethod === 'stripe'
      ? ''
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const airwallexRouteUrl = result.client_secret && result.intent_id
      ? router.resolve({
        path: '/payment/airwallex',
        query: {
          order_id: String(result.order_id),
          out_trade_no: result.out_trade_no || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType,
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && visibleMethod === 'alipay'),
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
      airwallexRouteUrl,
    })

    if (decision.kind === 'wechat_oauth' && decision.oauth?.authorize_url) {
      window.location.href = buildWechatOAuthAuthorizeUrl(decision.oauth.authorize_url, {
        paymentType: visibleMethod,
        orderType,
        planId,
        orderAmount,
      })
      return
    }

    if (decision.kind === 'unhandled') {
      applyScenarioError({ reason: 'UNHANDLED_PAYMENT_SCENARIO' }, visibleMethod)
      return
    }

    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)

    if (decision.kind === 'stripe_popup') {
      openWindow(decision.paymentState.payUrl)
      return
    }
    if (decision.kind === 'stripe_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'airwallex_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'wechat_jsapi' && decision.jsapi) {
      try {
        const jsapiResult = await invokeWechatJsapiPayment(decision.jsapi as Record<string, unknown>)
        const errMsg = String(jsapiResult.err_msg || '').toLowerCase()
        if (errMsg.includes('cancel')) {
          appStore.showInfo(t('payment.qr.cancelled'))
          resetPayment()
        } else if (errMsg && !errMsg.includes('ok')) {
          resetPayment()
          const fallbackApplied = await attemptMobileQrFallback(
            { reason: 'WECHAT_JSAPI_FAILED', message: errMsg },
            {
              orderAmount,
              orderType,
              planId,
              paymentType: visibleMethod,
              attempted: options.mobileQrFallbackAttempted === true,
              rechargePackageId: options.rechargePackageId,
            },
          )
          if (!fallbackApplied) {
            applyScenarioError({ reason: 'WECHAT_JSAPI_FAILED', message: errMsg }, visibleMethod)
          }
        } else {
          const resultState = { ...decision.paymentState }
          resetPayment()
          await redirectToPaymentResult(resultState)
        }
      } catch (err: unknown) {
        resetPayment()
        const fallbackApplied = await attemptMobileQrFallback(err, {
          orderAmount,
          orderType,
          planId,
          paymentType: visibleMethod,
          attempted: options.mobileQrFallbackAttempted === true,
          rechargePackageId: options.rechargePackageId,
        })
        if (!fallbackApplied) {
          throw err
        }
      }
      return
    }
    if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
      if (isMobileDevice()) {
        window.location.href = decision.paymentState.payUrl
        return
      }
      openWindow(decision.paymentState.payUrl)
    }
  } catch (err: unknown) {
    const apiErr = err as Record<string, unknown>
    if (apiErr.reason === 'TOO_MANY_PENDING') {
      const metadata = apiErr.metadata as Record<string, unknown> | undefined
      errorMessage.value = t('payment.errors.tooManyPending', { max: metadata?.max || '' })
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'CANCEL_RATE_LIMITED') {
      errorMessage.value = t('payment.errors.cancelRateLimited')
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'RECHARGE_PACKAGE_CHANGED' || apiErr.reason === 'RECHARGE_PACKAGE_DISABLED') {
      const refreshed = await paymentAPI.getCheckoutInfo()
      checkout.value = refreshed.data
      paymentStore.setBonusSummary(refreshed.data, user.value?.balance || 0)
      selectedRechargePackageId.value = ''
      errorMessage.value = t('payment.rechargePackages.changed')
      errorHintMessage.value = ''
    } else if (await attemptMobileQrFallback(err, {
      orderAmount,
      orderType,
      planId,
      paymentType: requestType,
      attempted: options.mobileQrFallbackAttempted === true,
      rechargePackageId: options.rechargePackageId,
    })) {
      return
    } else {
      const handled = applyScenarioError(
        err,
        normalizeVisibleMethod(options.paymentType || selectedMethod.value) || selectedMethod.value,
      )
      if (!handled) {
        errorMessage.value = extractI18nErrorMessage(err, t, 'payment.errors', extractApiErrorMessage(err, t('payment.result.failed')))
        errorHintMessage.value = ''
      }
      if (handled) {
        return
      }
    }
    appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  } finally {
    submitting.value = false
  }
}

interface MobileQrFallbackContext {
  orderAmount: number
  orderType: OrderType
  planId?: number
  paymentType: string
  attempted: boolean
  rechargePackageId?: string
}

function shouldFallbackToDesktopQr(err: unknown, paymentMethod: string, attempted: boolean): boolean {
  if (attempted || !isMobileDevice()) {
    return false
  }

  const normalizedMethod = normalizeVisibleMethod(paymentMethod) || paymentMethod
  const reason = typeof err === 'object' && err && 'reason' in err && typeof err.reason === 'string'
    ? err.reason
    : ''
  const message = err instanceof Error
    ? err.message
    : (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
      ? err.message
      : '')
  const normalizedMessage = message.toLowerCase()

  if (normalizedMethod === 'wxpay') {
    return reason === 'WECHAT_H5_NOT_AUTHORIZED'
      || reason === 'WECHAT_PAYMENT_MP_NOT_CONFIGURED'
      || reason === 'WECHAT_JSAPI_FAILED'
      || reason === 'PAYMENT_GATEWAY_ERROR'
      || reason === 'UNHANDLED_PAYMENT_SCENARIO'
      || normalizedMessage.includes('weixinjsbridge is unavailable')
      || normalizedMessage.includes('wechat_jsapi_unavailable')
  }

  if (normalizedMethod === 'alipay') {
    return reason === 'PAYMENT_GATEWAY_ERROR' || reason === 'UNHANDLED_PAYMENT_SCENARIO'
  }

  return false
}

async function attemptMobileQrFallback(err: unknown, context: MobileQrFallbackContext): Promise<boolean> {
  if (!shouldFallbackToDesktopQr(err, context.paymentType, context.attempted)) {
    return false
  }

  try {
    const visibleMethod = normalizeVisibleMethod(context.paymentType) || context.paymentType
    const payload = buildCreateOrderPayload({
      amount: context.orderAmount,
      paymentType: visibleMethod,
      orderType: context.orderType,
      planId: context.planId,
      rechargePackageId: context.rechargePackageId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: false,
      isWechatBrowser: false,
    })
    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const stripeMethod = visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType: context.orderType,
      isMobile: false,
      isWechatBrowser: false,
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
    })

    if (decision.kind !== 'qr_waiting' || !decision.paymentState.qrCode) {
      return false
    }

    errorMessage.value = ''
    errorHintMessage.value = ''
    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)
    appStore.showWarning(t('payment.errors.mobilePaymentFallbackToQr'))
    return true
  } catch {
    return false
  }
}

function applyScenarioError(err: unknown, paymentMethod: string): boolean {
  const descriptor = describePaymentScenarioError(err, {
    paymentMethod,
    isMobile: isMobileDevice(),
    isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
  })
  if (!descriptor) {
    errorMessage.value = ''
    errorHintMessage.value = ''
    return false
  }
  errorMessage.value = t(descriptor.messageKey)
  errorHintMessage.value = descriptor.hintKey ? t(descriptor.hintKey) : ''
  appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  return true
}

async function resumeWechatPaymentFromQuery() {
  const resume = parseWechatResumeRoute(route.query, checkout.value.plans, validAmount.value)
  if (!resume) {
    return
  }

  selectedMethod.value = resume.paymentType
  if (resume.orderType === 'balance' && resume.orderAmount > 0) {
    amount.value = resume.orderAmount
  }
  if (resume.orderType === 'subscription' && resume.planId) {
    selectedPlan.value = checkout.value.plans.find(plan => plan.id === resume.planId) ?? null
  }

  await router.replace({ path: route.path, query: stripWechatResumeQuery(route.query) })

  if (resume.wechatResumeToken) {
    await createOrder(0, resume.orderType, resume.planId, {
      wechatResumeToken: resume.wechatResumeToken,
      paymentType: resume.paymentType,
      isResume: true,
    })
    return
  }

  if (resume.orderAmount > 0 && resume.openid) {
    await createOrder(resume.orderAmount, resume.orderType, resume.planId, {
      openid: resume.openid,
      paymentType: resume.paymentType,
      isResume: true,
    })
  }
}

onMounted(async () => {
  try {
    const res = await paymentAPI.getCheckoutInfo()
    checkout.value = res.data
    paymentStore.setBonusSummary(res.data, user.value?.balance || 0)
    if (checkout.value.recharge_packages_enabled && rechargePackages.value.length > 0) {
      selectRechargePackage(rechargePackages.value.find(pkg => pkg.recommended) || rechargePackages.value[0])
    }
    if (enabledMethods.value.length) {
      const order: readonly string[] = METHOD_ORDER
      const sorted = [...enabledMethods.value].sort((a, b) => {
        const ai = order.indexOf(a)
        const bi = order.indexOf(b)
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
      })
      selectedMethod.value = sorted[0]
    }
    if (typeof window !== 'undefined') {
      if (hasWechatResumeQuery(route.query)) {
        removeRecoverySnapshot()
      }
      const routeResumeToken = typeof route.query.resume_token === 'string'
        ? route.query.resume_token
        : typeof route.query.wechat_resume_token === 'string'
          ? route.query.wechat_resume_token
          : undefined
      const restored = readPaymentRecoverySnapshot(
        window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
        { resumeToken: routeResumeToken },
      )
      if (restored) {
        paymentState.value = restored
        paymentPhase.value = 'paying'
        const restoredMethod = normalizeVisibleMethod(restored.paymentType)
          || (visibleMethods.value[restored.paymentType] ? restored.paymentType : '')
        if (restoredMethod) {
          selectedMethod.value = restoredMethod
        }
        if (restored.rechargePackageId) {
          selectedRechargePackageId.value = restored.rechargePackageId
        }
      } else {
        removeRecoverySnapshot()
      }
    }
    await resumeWechatPaymentFromQuery()
    if (checkout.value.balance_disabled) {
      activeTab.value = 'subscription'
    }
    // Handle renewal navigation: ?tab=subscription&group=123
    if (route.query.tab === 'subscription') {
      activeTab.value = 'subscription'
      if (route.query.group) {
        const groupId = Number(route.query.group)
        const groupPlans = checkout.value.plans.filter(p => p.group_id === groupId)
        if (groupPlans.length === 1) {
          selectedPlan.value = groupPlans[0]
        } else if (groupPlans.length > 1) {
          renewGroupId.value = groupId
          showRenewalModal.value = true
        }
      }
    }
  } catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { loading.value = false }
  // Fetch active subscriptions (uses cache, non-blocking)
  subscriptionStore.fetchActiveSubscriptions().catch(() => {})
})
</script>

<style scoped>
.recharge-checkout-shell {
  border: 1px solid #d7e5fa;
  border-radius: 22px;
  background:
    linear-gradient(135deg, rgb(239 246 255 / 0.96), rgb(240 253 250 / 0.78)),
    #f8fbff;
  padding: 26px 32px 32px;
  box-shadow: 0 14px 36px rgb(30 64 175 / 0.09);
}

.recharge-cashier-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 24px;
  border-bottom: 1px solid #dbe7f5;
  padding-bottom: 24px;
}

.recharge-cashier-header h1 {
  color: #0f172a;
  font-size: 22px;
  font-weight: 750;
  line-height: 30px;
}

.recharge-cashier-header p {
  margin-top: 2px;
  color: #64748b;
  font-size: 13px;
  line-height: 20px;
}

.recharge-cashier-icon {
  display: inline-flex;
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: #2563eb;
  color: #fff;
  box-shadow: 0 8px 18px rgb(37 99 235 / 0.22);
}

.recharge-security-badge {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  border: 1px solid #a7f3d0;
  border-radius: 999px;
  background: #ecfdf5;
  padding: 9px 15px;
  color: #047857;
  font-size: 13px;
  font-weight: 600;
}

.recharge-panel,
.recharge-summary-panel,
.recharge-account-panel {
  border: 1px solid #dbe3ef;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04);
}

.recharge-panel {
  padding: 20px;
}

.recharge-summary-panel,
.recharge-account-panel {
  padding: 20px;
}

.recharge-step-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.recharge-step-header h2 {
  color: #0f172a;
  font-size: 17px;
  font-weight: 700;
  line-height: 24px;
}

.recharge-step-header p {
  margin-top: 2px;
  color: #64748b;
  font-size: 13px;
  line-height: 20px;
}

.recharge-step-number {
  display: inline-flex;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #2563eb;
  color: #fff;
  font-size: 14px;
  font-weight: 700;
}

.recharge-package-card {
  position: relative;
  min-height: 138px;
  overflow: hidden;
  border: 1px solid #dbe3ef;
  border-radius: 8px;
  background: #fff;
  padding: 18px 16px 14px;
  text-align: left;
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
}

.recharge-package-card:hover {
  border-color: #93b4f8;
  box-shadow: 0 5px 18px rgb(37 99 235 / 0.08);
  transform: translateY(-1px);
}

.recharge-package-card.is-selected {
  border-color: #2563eb;
  box-shadow: 0 0 0 1px #2563eb, 0 6px 18px rgb(37 99 235 / 0.12);
}

.recharge-recommended {
  position: absolute;
  top: 10px;
  right: 10px;
  border: 1px solid #fed7aa;
  border-radius: 999px;
  background: #fff7ed;
  padding: 2px 8px;
  color: #ea580c;
  font-size: 10px;
  font-weight: 600;
}

.recharge-bonus-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 14px;
  border-radius: 999px;
  background: #ecfdf5;
  padding: 5px 9px;
  color: #047857;
  font-size: 11px;
  font-weight: 600;
}

.recharge-bonus-notice {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  border: 1px solid #a7f3d0;
  border-radius: 10px;
  background: #ecfdf5;
  padding: 13px 16px;
  color: #047857;
  font-size: 12px;
  line-height: 20px;
}

.recharge-selected-check {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #2563eb;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.recharge-help-panel {
  border: 1px solid #f4c95d;
  border-radius: 12px;
  background: #fffbeb;
  padding: 20px;
  box-shadow: 0 4px 12px rgb(180 83 9 / 0.04);
}

.recharge-contact-button {
  display: inline-flex;
  width: 100%;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
  border: 1px solid #f4c95d;
  border-radius: 10px;
  background: rgb(255 255 255 / 0.72);
  padding: 10px 14px;
  color: #b45309;
  font-size: 13px;
  font-weight: 600;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.recharge-contact-button:hover {
  border-color: #f59e0b;
  background: #fff;
}

.recharge-risk-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 1px solid #e5e7eb;
  border-radius: 14px;
  background: #fff;
  padding: 16px 22px;
  color: #64748b;
  font-size: 13px;
  line-height: 20px;
  text-align: center;
  box-shadow: 0 2px 8px rgb(15 23 42 / 0.04);
}

html.dark .recharge-checkout-shell {
  border-color: #334155;
  background: linear-gradient(135deg, #111827, #0f2629);
  box-shadow: 0 14px 36px rgb(0 0 0 / 0.22);
}

html.dark .recharge-cashier-header {
  border-color: #334155;
}

html.dark .recharge-cashier-header h1 {
  color: #f8fafc;
}

html.dark .recharge-cashier-header p,
html.dark .recharge-step-header p {
  color: #94a3b8;
}

html.dark .recharge-security-badge {
  border-color: #065f46;
  background: rgb(6 78 59 / 0.42);
  color: #6ee7b7;
}

html.dark .recharge-panel,
html.dark .recharge-summary-panel,
html.dark .recharge-account-panel,
html.dark .recharge-package-card {
  border-color: #334155;
  background: #111827;
}

html.dark .recharge-step-header h2 {
  color: #f8fafc;
}

html.dark .recharge-package-card:hover {
  border-color: #475569;
  background: #172033;
  box-shadow: 0 5px 18px rgb(0 0 0 / 0.2);
}

html.dark .recharge-package-card.is-selected {
  border-color: #60a5fa;
  box-shadow: 0 0 0 1px #60a5fa;
}

html.dark .recharge-recommended {
  border-color: #7c2d12;
  background: #431407;
  color: #fdba74;
}

html.dark .recharge-bonus-pill {
  background: rgb(6 78 59 / 0.5);
  color: #6ee7b7;
}

html.dark .recharge-bonus-notice {
  border-color: #065f46;
  background: rgb(6 78 59 / 0.36);
  color: #a7f3d0;
}

html.dark .recharge-help-panel {
  border-color: #854d0e;
  background: rgb(69 26 3 / 0.35);
}

html.dark .recharge-contact-button {
  border-color: #92400e;
  background: rgb(120 53 15 / 0.22);
  color: #fdba74;
}

html.dark .recharge-contact-button:hover {
  background: rgb(120 53 15 / 0.4);
}

html.dark .recharge-risk-footer {
  border-color: #334155;
  background: #111827;
  color: #94a3b8;
}

@media (max-width: 1023px) {
  .recharge-workspace {
    padding-bottom: 76px;
  }
}

@media (max-width: 639px) {
  .recharge-checkout-shell {
    border-radius: 14px;
    padding: 18px 14px 20px;
  }

  .recharge-cashier-header {
    align-items: flex-start;
    margin-bottom: 18px;
    padding-bottom: 18px;
  }

  .recharge-cashier-icon {
    width: 42px;
    height: 42px;
    flex-basis: 42px;
  }

  .recharge-security-badge {
    padding: 7px 10px;
    font-size: 11px;
  }

  .recharge-risk-footer {
    align-items: flex-start;
    padding: 14px 16px;
    text-align: left;
  }

  .recharge-panel,
  .recharge-summary-panel,
  .recharge-account-panel {
    padding: 16px;
  }

  .recharge-package-card {
    min-height: 132px;
    padding: 16px 12px 12px;
  }
}
</style>
