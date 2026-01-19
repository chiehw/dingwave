import { ref } from 'vue'

export function useSearchDebounce(callback: (keyword: string) => void, delay = 300) {
  const searchKeyword = ref('')
  const isComposing = ref(false)
  let searchTimer: ReturnType<typeof setTimeout> | null = null

  const debouncedSearch = () => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      callback(searchKeyword.value.trim())
    }, delay)
  }

  const onInputChange = () => {
    if (!isComposing.value) {
      debouncedSearch()
    }
  }

  const onCompositionEnd = () => {
    isComposing.value = false
    debouncedSearch()
  }

  const clearSearch = () => {
    searchKeyword.value = ''
    if (searchTimer) clearTimeout(searchTimer)
  }

  return {
    searchKeyword,
    isComposing,
    onInputChange,
    onCompositionEnd,
    clearSearch,
  }
}
