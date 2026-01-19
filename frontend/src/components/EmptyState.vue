<template>
  <div class="empty-tip">
    {{ message }}
  </div>
</template>

<script setup lang="ts">
defineProps<{
  message: string
}>()
</script>

<style scoped>
.empty-tip {
  text-align: center;
  padding: 24px;
  color: rgba(128, 128, 128, 0.6);
}
</style>
