export interface User {
  id: number
  nickname: string
  email: string
}

export interface Conversation {
  id: number
  cid: string
  type: number
  title: string
  is_top: boolean
  message_count: number
  last_message_at: number
  last_message_id: number
  last_message_preview: string
  created_at: number
}

export interface ConversationList {
  total: number
  items: Conversation[]
}

export interface HomeResponse {
  current_user: User
  top: ConversationList
  single: ConversationList
  group: ConversationList
}

export interface Message {
  id: number
  cid: string
  sender_id: number
  sender_name: string
  content_type: number
  content_text: string
  content_json: string
  created_at: number
  is_recall: boolean
}

export interface MessagesResponse {
  has_more: boolean
  items: Message[]
}

export interface PageResponse<T> {
  total: number
  page: number
  size: number
  items: T[]
}

export interface SearchAggregatedItem {
  cid: string
  title: string
  type: number
  match_count: number
}

export interface SearchMessageItem extends Omit<Message, 'content_json'> {}

async function apiFetch<T>(url: string): Promise<T> {
  const response = await fetch(url)
  return response.json()
}

export function fetchHome(limit = 5): Promise<HomeResponse> {
  return apiFetch(`/api/conversations/home?limit=${limit}`)
}

export function fetchConversations(type: number, page = 1, size = 20): Promise<PageResponse<Conversation>> {
  return apiFetch(`/api/conversations?type=${type}&page=${page}&size=${size}`)
}

export function fetchMessages(cid: string, before?: number): Promise<MessagesResponse> {
  const url = before ? `/api/conversations/${cid}/messages?before=${before}` : `/api/conversations/${cid}/messages`
  return apiFetch(url)
}

export function fetchMessagesAfter(cid: string, after: number, size = 50): Promise<MessagesResponse> {
  return apiFetch(`/api/conversations/${cid}/messages?after=${after}&size=${size}`)
}

export function searchMessages(keyword: string, page = 1, size = 20): Promise<PageResponse<SearchAggregatedItem>> {
  return apiFetch(`/api/messages/search?q=${encodeURIComponent(keyword)}&page=${page}&size=${size}`)
}

export function searchConversationMessages(cid: string, keyword: string, page = 1, size = 20): Promise<PageResponse<SearchMessageItem>> {
  return apiFetch(`/api/conversations/${cid}/messages/search?q=${encodeURIComponent(keyword)}&page=${page}&size=${size}`)
}

export function fetchUsers(page = 1, size = 50): Promise<PageResponse<User>> {
  return apiFetch(`/api/users?page=${page}&size=${size}`)
}

export function searchUsers(keyword: string): Promise<User[]> {
  return apiFetch(`/api/users/search?q=${encodeURIComponent(keyword)}`)
}

export function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function highlightText(text: string, keyword: string): string {
  if (!keyword || !text) return text
  const escaped = escapeRegex(keyword)
  const regex = new RegExp(`(${escaped})`, 'gi')
  return text.replace(regex, '<mark class="search-highlight">$1</mark>')
}

export function createConversationFromSearch(item: SearchAggregatedItem): Conversation {
  return {
    id: 0,
    cid: item.cid,
    type: item.type,
    title: item.title,
    is_top: false,
    message_count: 0,
    last_message_at: 0,
    last_message_id: 0,
    last_message_preview: '',
    created_at: 0,
  }
}
