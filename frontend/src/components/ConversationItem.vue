<template>
  <div :class="['conv-item', { active: isActive }]" @click="$emit('click')">
    <a-avatar :size="40" :style="{ backgroundColor: getConversationAvatarColor(conversation.type) }">
      {{ conversation.title?.charAt(0) }}
    </a-avatar>
    <div class="conv-info">
      <div class="conv-title">{{ conversation.title }}</div>
      <div class="conv-preview">{{ preview }}</div>
    </div>
    <div v-if="showTime" class="conv-time">{{ formatTimestamp(conversation.last_message_at) }}</div>
    <RightOutlined v-if="showArrow" class="section-arrow" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RightOutlined } from '@ant-design/icons-vue'
import type { Conversation } from '@/api'
import { getConversationAvatarColor } from '@/utils/avatar'
import { formatTimestamp } from '@/utils/time'

const props = defineProps<{
  conversation: Conversation
  isActive?: boolean
  showTime?: boolean
  showArrow?: boolean
}>()

defineEmits<{
  click: []
}>()

const preview = computed(() => props.conversation.last_message_preview)
</script>

<style scoped>
.conv-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  gap: 10px;
}
.conv-item:hover {
  background: rgba(128, 128, 128, 0.1);
}
.conv-item.active {
  background: rgba(22, 119, 255, 0.1);
}
.conv-info {
  flex: 1;
  min-width: 0;
}
.conv-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.conv-preview {
  font-size: 12px;
  color: rgba(128, 128, 128, 0.8);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.conv-time {
  font-size: 12px;
  color: rgba(128, 128, 128, 0.6);
  white-space: nowrap;
  padding-right: 4px;
}
.section-arrow {
  font-size: 12px;
  color: rgba(128, 128, 128, 0.6);
}
</style>
