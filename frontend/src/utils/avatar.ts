export function getConversationAvatarColor(type: number): string {
  return type === 2 ? '#87d068' : '#1677ff'
}

export function getMessageAvatarColor(isMine: boolean): string {
  return isMine ? '#52c41a' : '#1677ff'
}
