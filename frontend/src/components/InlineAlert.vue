<template>
  <div v-if="message" class="inline-alert" :class="variantClass">
    {{ message }}
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  message: {
    type: String,
    default: "",
  },
  variant: {
    type: String,
    default: "info",
  },
});

// variantClass 输入组件 props，输出为样式类，目的在于统一提示框成功和错误态的视觉表现。
const variantClass = computed(() => {
  if (props.variant === "success") {
    return "is-success";
  }
  if (props.variant === "error") {
    return "is-error";
  }
  return "";
});
</script>
