<template>
  <div class="join-group-anchor">
    <Transition name="join-group-button">
      <div
        v-if="isExpanded"
        class="join-group-content"
        @mouseenter="showPopup = true"
        @mouseleave="showPopup = false"
      >
        <Transition name="join-group-popup">
          <section v-if="showPopup" class="join-group-popup" aria-label="加入交流群">
            <div class="join-group-heading">
              <span class="join-group-heading-icon"><Icon name="users" size="sm" /></span>
              <div>
                <p class="join-group-title">加入交流群</p>
                <p class="join-group-subtitle">扫码加入交流群</p>
              </div>
            </div>
            <img :src="qrCodeSrc" alt="交流群二维码" class="join-group-qr" />
            <button type="button" class="join-group-action" @click="openGroupLink">
              点击加群
            </button>
            <p class="join-group-hint">扫码或点击加入交流群，获取使用通知和交流支持。</p>
          </section>
        </Transition>

        <button type="button" class="join-group-trigger" aria-label="加入交流群" @click="openGroupLink">
          <Icon name="users" size="sm" />
          <span>加群</span>
        </button>
      </div>
    </Transition>
    <button type="button" class="join-group-chevron" :aria-label="isExpanded ? '隐藏加群按钮' : '显示加群按钮'" @click="toggleExpanded">
      <Icon name="chevronLeft" size="xs" :class="isExpanded ? 'rotate-180' : ''" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import chatQrCode from '@/assets/chat_qrcode.jpg'

const groupLink = 'https://qm.qq.com/q/HMENjBu4YA'
const qrCodeSrc = chatQrCode
const isExpanded = ref(true)
const showPopup = ref(false)

function toggleExpanded() {
  isExpanded.value = !isExpanded.value
  showPopup.value = false
}

function openGroupLink() {
  window.open(groupLink, '_blank', 'noopener,noreferrer')
}
</script>

<style scoped>
.join-group-anchor {
  position: fixed;
  right: 0;
  bottom: 28px;
  z-index: 50;
  width: 110px;
  height: 64px;
}

.join-group-content {
  position: absolute;
  right: 39px;
  bottom: 0;
}

.join-group-popup {
  position: absolute;
  right: 0;
  bottom: 64px;
  width: 246px;
  border: 1px solid #c9e9f8;
  border-radius: 14px;
  background: #fff;
  padding: 14px;
  box-shadow: 0 18px 42px rgb(15 23 42 / 0.2);
}

.join-group-popup::after {
  position: absolute;
  right: 19px;
  bottom: -8px;
  width: 16px;
  height: 16px;
  border-right: 1px solid #c9e9f8;
  border-bottom: 1px solid #c9e9f8;
  background: #fff;
  content: '';
  transform: rotate(45deg);
}

.join-group-heading {
  display: flex;
  align-items: center;
  gap: 9px;
}

.join-group-heading-icon {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 1px solid #a8e3f7;
  border-radius: 8px;
  background: #effaff;
  color: #0ea5e9;
}

.join-group-title {
  color: #0f172a;
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
}

.join-group-subtitle,
.join-group-hint {
  color: #64748b;
  font-size: 11px;
  line-height: 17px;
}

.join-group-qr {
  display: block;
  width: 218px;
  height: 218px;
  margin: 10px auto;
  border-radius: 10px;
  object-fit: cover;
}

.join-group-action {
  width: 100%;
  border: 1px solid #a8e3f7;
  border-radius: 9px;
  background: #effaff;
  padding: 8px 12px;
  color: #0284c7;
  font-size: 12px;
  font-weight: 700;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.join-group-action:hover {
  border-color: #38bdf8;
  background: #e0f7ff;
}

.join-group-hint {
  margin-top: 9px;
  font-weight: 600;
}

.join-group-trigger {
  display: inline-flex;
  width: 70px;
  height: 64px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid #c9e9f8;
  border-radius: 16px;
  background: #fff;
  color: #075985;
  font-size: 12px;
  font-weight: 700;
  box-shadow: 0 8px 22px rgb(15 23 42 / 0.12);
}

.join-group-trigger:hover {
  background: #f0fbff;
}

.join-group-chevron {
  position: absolute;
  top: 7px;
  right: 0;
  display: inline-flex;
  width: 25px;
  height: 50px;
  align-items: center;
  justify-content: center;
  border: 1px solid #c9e9f8;
  border-right: 0;
  border-radius: 999px 0 0 999px;
  background: #fff;
  color: #075985;
  box-shadow: 6px 8px 22px rgb(15 23 42 / 0.08);
}

.join-group-chevron :deep(svg) {
  transition: transform 160ms ease;
}

.join-group-popup-enter-active,
.join-group-popup-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.join-group-popup-enter-from,
.join-group-popup-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.97);
}

.join-group-button-enter-active,
.join-group-button-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.join-group-button-enter-from,
.join-group-button-leave-to {
  opacity: 0;
  transform: translateX(8px);
}

@media (max-width: 639px) {
  .join-group-anchor {
    right: 0;
    bottom: 84px;
  }

  .join-group-popup {
    right: 0;
    width: min(246px, calc(100vw - 28px));
  }
}
</style>
