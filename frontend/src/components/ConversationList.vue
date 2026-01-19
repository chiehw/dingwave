<template>
  <div class="conversation-list">
    <!-- 搜索框 -->
    <div class="search-box">
      <a-input
        v-model:value="searchKeyword"
        placeholder="搜索聊天记录"
        allow-clear
        @compositionstart="isComposing = true"
        @compositionend="onCompositionEnd"
        @input="onInputChange"
        @clear="exitSearch"
      >
        <template #prefix>
          <SearchOutlined style="color: rgba(128, 128, 128, 0.6)" />
        </template>
      </a-input>
    </div>

    <!-- 固定的返回按钮区域 -->
    <div v-if="viewMode === 'search-aggregated'" class="sticky-header" @click="exitSearch">
      <LeftOutlined class="back-arrow" />
      <span class="header-title">搜索结果</span>
    </div>
    <div v-else-if="viewMode === 'search-conversation'" class="sticky-header" @click="backToAggregated">
      <LeftOutlined class="back-arrow" />
      <span class="header-title">{{ currentSearchConversation?.title }}</span>
    </div>
    <div v-else-if="expandedType !== null" class="sticky-header" @click="collapse">
      <LeftOutlined class="back-arrow" />
      <span class="header-title">{{ expandLabel }}</span>
    </div>

    <div class="list-content" @scroll="onScroll">
      <a-spin :spinning="loading || searchLoading" class="spin-container">
        <!-- 搜索聚合结果模式 -->
        <template v-if="viewMode === 'search-aggregated'">
          <EmptyState v-if="searchResults.length === 0 && !searchLoading" message="未找到相关消息" />
          <div
            v-for="item in searchResults"
            :key="item.cid"
            class="conv-item"
            @click="enterConversationSearch(item)"
          >
            <a-avatar :size="40" :style="{ backgroundColor: item.type === 2 ? '#87d068' : '#1677ff' }">
              {{ item.title?.charAt(0) }}
            </a-avatar>
            <div class="conv-info">
              <div class="conv-title">{{ item.title }}</div>
              <div class="conv-preview">{{ item.match_count }} 条相关消息</div>
            </div>
            <RightOutlined class="section-arrow" />
          </div>
        </template>

        <!-- 会话内搜索结果模式 -->
        <template v-else-if="viewMode === 'search-conversation'">
          <div
            v-for="msg in conversationSearchResults"
            :key="msg.id"
            class="message-result-item"
            @click="jumpToMessage(msg)"
          >
            <div class="msg-sender">{{ msg.sender_name }}</div>
            <div class="msg-content" v-html="highlightKeyword(msg.content_text)"></div>
            <div class="msg-time">{{ formatTimestamp(msg.created_at) }}</div>
          </div>
          <div v-if="convSearchHasMore" class="load-more-spinner">
            <a-spin size="small" />
          </div>
        </template>

        <!-- 展开模式 -->
        <template v-else-if="expandedType !== null">
          <ConversationItem
            v-for="item in expandedList"
            :key="item.cid"
            :conversation="item"
            :is-active="item.cid === selectedCid"
            :show-time="true"
            @click="$emit('select', item)"
          />
          <div v-if="expandHasMore" class="load-more-spinner">
            <a-spin size="small" />
          </div>
        </template>

        <!-- 默认分组模式 -->
        <template v-else>
          <div v-for="section in sections" :key="section.type" class="section">
            <div class="section-header" @click="expand(section.type)">
              <span class="section-title">{{ section.label }}</span>
              <span class="section-count">{{ section.data?.total || 0 }}</span>
              <RightOutlined class="section-arrow" />
            </div>
            <ConversationItem
              v-for="item in section.data?.items || []"
              :key="item.cid"
              :conversation="item"
              :is-active="item.cid === selectedCid"
              :show-time="true"
              @click="$emit('select', item)"
            />
          </div>
        </template>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { RightOutlined, LeftOutlined, SearchOutlined } from '@ant-design/icons-vue'
import {
  fetchConversations,
  searchMessages,
  searchConversationMessages,
  highlightText,
  type HomeResponse,
  type Conversation,
  type SearchAggregatedItem,
  type SearchMessageItem,
} from '@/api'
import { formatTimestamp } from '@/utils/time'
import { useSearchDebounce } from '@/composables/useSearchDebounce'
import ConversationItem from './ConversationItem.vue'
import EmptyState from './EmptyState.vue'

const props = defineProps<{
  homeData: HomeResponse | null
  selectedCid: string
  loading: boolean
}>()

const emit = defineEmits<{
  select: [item: Conversation]
  jumpToMessage: [cid: string, messageId: number, keyword: string, conversation: SearchAggregatedItem, timestamp: number]
}>()

// 视图模式
type ViewMode = 'default' | 'expanded' | 'search-aggregated' | 'search-conversation'
const viewMode = ref<ViewMode>('default')

// 搜索状态
const searchResults = ref<SearchAggregatedItem[]>([])
const conversationSearchResults = ref<SearchMessageItem[]>([])
const currentSearchConversation = ref<SearchAggregatedItem | null>(null)
const searchLoading = ref(false)
const convSearchPage = ref(1)
const convSearchTotal = ref(0)

// 搜索相关方法
const doSearch = async (keyword: string) => {
  if (!keyword) {
    exitSearch()
    return
  }
  searchLoading.value = true
  viewMode.value = 'search-aggregated'
  const res = await searchMessages(keyword)
  searchResults.value = res.items
  searchLoading.value = false
}

const { searchKeyword, isComposing, onInputChange, onCompositionEnd, clearSearch: clearSearchInput } = useSearchDebounce(doSearch)

// 展开模式状态
const expandedType = ref<number | null>(null)
const expandedList = ref<Conversation[]>([])
const expandPage = ref(1)
const expandTotal = ref(0)
const expandLoading = ref(false)

const sections = computed(() => [
  { type: 0, label: '置顶', data: props.homeData?.top },
  { type: 1, label: '单聊', data: props.homeData?.single },
  { type: 2, label: '群聊', data: props.homeData?.group },
])

const expandLabel = computed(() => {
  const labels: Record<number, string> = { 0: '置顶', 1: '单聊', 2: '群聊' }
  return labels[expandedType.value!] || ''
})

const expandHasMore = computed(() => expandedList.value.length < expandTotal.value)

const enterConversationSearch = async (item: SearchAggregatedItem) => {
  currentSearchConversation.value = item
  searchLoading.value = true
  viewMode.value = 'search-conversation'
  convSearchPage.value = 1
  const res = await searchConversationMessages(item.cid, searchKeyword.value, 1)
  conversationSearchResults.value = res.items
  convSearchTotal.value = res.total
  searchLoading.value = false
}

const convSearchHasMore = computed(() => conversationSearchResults.value.length < convSearchTotal.value)

const loadMoreConvSearch = async () => {
  if (searchLoading.value || !convSearchHasMore.value || !currentSearchConversation.value) return
  searchLoading.value = true
  const res = await searchConversationMessages(
    currentSearchConversation.value.cid,
    searchKeyword.value,
    ++convSearchPage.value
  )
  conversationSearchResults.value = [...conversationSearchResults.value, ...res.items]
  searchLoading.value = false
}

const backToAggregated = () => {
  viewMode.value = 'search-aggregated'
  conversationSearchResults.value = []
  currentSearchConversation.value = null
}

const exitSearch = () => {
  viewMode.value = 'default'
  clearSearchInput()
  searchResults.value = []
  conversationSearchResults.value = []
  currentSearchConversation.value = null
}

const jumpToMessage = (msg: SearchMessageItem) => {
  emit('jumpToMessage', msg.cid, msg.id, searchKeyword.value, currentSearchConversation.value!, msg.created_at)
}

const highlightKeyword = (text: string) => highlightText(text, searchKeyword.value)

// 展开模式方法
const expand = async (type: number) => {
  expandedType.value = type
  viewMode.value = 'expanded'
  expandPage.value = 1
  expandLoading.value = true
  const res = await fetchConversations(type, 1)
  expandedList.value = res.items
  expandTotal.value = res.total
  expandLoading.value = false
}

const loadMoreExpand = async () => {
  if (expandLoading.value || !expandHasMore.value) return
  expandLoading.value = true
  const res = await fetchConversations(expandedType.value!, ++expandPage.value)
  expandedList.value = [...expandedList.value, ...res.items]
  expandLoading.value = false
}

const onScroll = (e: Event) => {
  const el = e.target as HTMLElement
  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 50
  if (!nearBottom) return

  if (viewMode.value === 'expanded') {
    loadMoreExpand()
  } else if (viewMode.value === 'search-conversation') {
    loadMoreConvSearch()
  }
}

const collapse = () => {
  expandedType.value = null
  viewMode.value = 'default'
  expandedList.value = []
}
</script>

<style scoped>
.conversation-list {
  width: 280px;
  height: 100%;
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgba(128, 128, 128, 0.2);
}
.search-box {
  padding: 12px;
  border-bottom: 1px solid rgba(128, 128, 128, 0.2);
}
.list-content {
  flex: 1;
  overflow-y: auto;
}
.section {
  margin-bottom: 8px;
}
.section-header {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  font-weight: 500;
}
.section-header:hover {
  background: rgba(128, 128, 128, 0.1);
}
.section-title {
  flex: 1;
}
.section-count {
  color: rgba(128, 128, 128, 0.6);
  font-size: 12px;
  margin-right: 8px;
}
.section-arrow {
  font-size: 12px;
  color: rgba(128, 128, 128, 0.6);
}
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
.sticky-header {
  display: flex;
  align-items: center;
  padding: 12px;
  cursor: pointer;
  font-weight: 500;
  gap: 8px;
  border-bottom: 1px solid rgba(128, 128, 128, 0.2);
  flex-shrink: 0;
}
.sticky-header:hover {
  background: rgba(128, 128, 128, 0.1);
}
.back-arrow {
  font-size: 14px;
  color: #1677ff;
}
.header-title {
  color: #1677ff;
}
.load-more-spinner {
  text-align: center;
  padding: 12px;
}
.empty-tip {
  text-align: center;
  padding: 24px;
  color: rgba(128, 128, 128, 0.6);
}
.message-result-item {
  padding: 10px 12px;
  cursor: pointer;
  border-bottom: 1px solid rgba(128, 128, 128, 0.1);
}
.message-result-item:hover {
  background: rgba(128, 128, 128, 0.1);
}
.msg-sender {
  font-weight: 500;
  font-size: 13px;
  margin-bottom: 4px;
}
.msg-content {
  font-size: 13px;
  color: rgba(128, 128, 128, 0.9);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.msg-time {
  font-size: 11px;
  color: rgba(128, 128, 128, 0.6);
  margin-top: 4px;
}
:deep(.search-highlight) {
  background-color: #ffe58f;
  padding: 0 2px;
  border-radius: 2px;
}
</style>
