export function useScrollPagination(
  onLoadMore?: () => void,
  onLoadMoreAfter?: () => void,
  threshold = 50
) {
  const onScroll = (e: Event) => {
    const el = e.target as HTMLElement

    if (onLoadMore && el.scrollTop < threshold) {
      onLoadMore()
    }

    if (onLoadMoreAfter && el.scrollTop + el.clientHeight >= el.scrollHeight - threshold) {
      onLoadMoreAfter()
    }
  }

  return {
    onScroll,
  }
}
