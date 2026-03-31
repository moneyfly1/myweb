<template>
  <div class="animated-character-wrapper">
    <svg
      viewBox="0 0 200 220"
      xmlns="http://www.w3.org/2000/svg"
      class="character-svg"
    >
      <!-- 地面阴影 -->
      <ellipse cx="100" cy="215" rx="52" ry="7" fill="rgba(0,0,0,0.07)"/>

      <!-- 左臂（头部后面渲染） -->
      <g :style="leftArmStyle">
        <ellipse cx="44" cy="170" rx="20" ry="13" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5" transform="rotate(-20 44 170)"/>
        <circle cx="30" cy="162" r="10" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
        <circle cx="23" cy="173" r="9.5" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
        <circle cx="30" cy="183" r="9" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
      </g>

      <!-- 右臂（头部后面渲染） -->
      <g :style="rightArmStyle">
        <ellipse cx="156" cy="170" rx="20" ry="13" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5" transform="rotate(20 156 170)"/>
        <circle cx="170" cy="162" r="10" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
        <circle cx="177" cy="173" r="9.5" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
        <circle cx="170" cy="183" r="9" fill="#FDDBB4" stroke="#ECC8A0" stroke-width="1.5"/>
      </g>

      <!-- 耳朵 -->
      <circle cx="58" cy="62" r="24" fill="#FDDBB4"/>
      <circle cx="58" cy="62" r="14" fill="#F4A5A5"/>
      <circle cx="142" cy="62" r="24" fill="#FDDBB4"/>
      <circle cx="142" cy="62" r="14" fill="#F4A5A5"/>

      <!-- 头部 -->
      <circle cx="100" cy="118" r="72" fill="#FDDBB4"/>

      <!-- 眼白 -->
      <circle cx="78" cy="108" r="17" fill="white"/>
      <circle cx="122" cy="108" r="17" fill="white"/>

      <!-- 瞳孔（跟踪用户名输入） -->
      <circle :cx="leftPupilX" :cy="pupilY" r="9" fill="#222"
        :style="{ transition: 'cx 0.1s ease-out' }"/>
      <circle :cx="leftPupilX + 3" :cy="pupilY - 3" r="3" fill="white"
        :style="{ transition: 'cx 0.1s ease-out' }"/>
      <circle :cx="rightPupilX" :cy="pupilY" r="9" fill="#222"
        :style="{ transition: 'cx 0.1s ease-out' }"/>
      <circle :cx="rightPupilX + 3" :cy="pupilY - 3" r="3" fill="white"
        :style="{ transition: 'cx 0.1s ease-out' }"/>

      <!-- 鼻子 -->
      <ellipse cx="100" cy="125" rx="6" ry="4" fill="#E8907A"/>

      <!-- 嘴巴 -->
      <path :d="mouthPath" fill="none" stroke="#C87060" stroke-width="2.5" stroke-linecap="round"
        style="transition: d 0.3s ease"/>

      <!-- 腮红 -->
      <ellipse cx="62" cy="128" rx="13" ry="8" fill="#FF9BAC" opacity="0.4"/>
      <ellipse cx="138" cy="128" rx="13" ry="8" fill="#FF9BAC" opacity="0.4"/>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  usernameValue: { type: String, default: '' },
  isPasswordFocused: { type: Boolean, default: false },
  isPasswordVisible: { type: Boolean, default: false }
})

// --- 眼球跟踪 ---
// 将用户名长度（0~25字符）映射为瞳孔偏移
const eyeRatio = computed(() => Math.min(props.usernameValue.length / 25, 1))
// 左眼中心 (78,108)，瞳孔在眼白内从左到右移动 8px
const leftPupilX = computed(() => 74 + eyeRatio.value * 8)
// 右眼中心 (122,108)
const rightPupilX = computed(() => 118 + eyeRatio.value * 8)
const pupilY = 108

// --- 嘴巴形状 ---
const mouthPath = computed(() => {
  if (props.isPasswordFocused && !props.isPasswordVisible) {
    return 'M 88 132 Q 100 132 112 132' // 紧张：直线
  }
  return 'M 88 130 Q 100 139 112 130' // 开心：微笑
})

// --- 手臂动画 ---
// 左臂组合的参考坐标约在 (30~44, 162~183)
// 眼睛在 y=108，捂眼需要向上移动约 64px，向右移动约 50px
const leftArmStyle = computed(() => ({
  transform: props.isPasswordFocused
    ? 'translate(50px, -64px)'
    : 'translate(0px, 0px)',
  transition: 'transform 0.55s cubic-bezier(0.34, 1.56, 0.64, 1)'
}))

const rightArmStyle = computed(() => {
  let ty = '-64px'
  if (props.isPasswordVisible) ty = '-35px' // 偷看：右臂下滑，露出右眼
  return {
    transform: props.isPasswordFocused
      ? `translate(-50px, ${ty})`
      : 'translate(0px, 0px)',
    transition: 'transform 0.55s cubic-bezier(0.34, 1.56, 0.64, 1)'
  }
})
</script>

<style scoped>
.animated-character-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  user-select: none;
  pointer-events: none;
}

.character-svg {
  width: 140px;
  height: 154px;
  animation: breathe 3.5s ease-in-out infinite;
  transform-origin: center bottom;
  overflow: visible;
}

@keyframes breathe {
  0%, 100% { transform: scaleY(1) translateY(0); }
  50%       { transform: scaleY(1.025) translateY(-2px); }
}
</style>
