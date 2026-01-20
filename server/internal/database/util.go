package database

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"dingtalk/internal/logger"
)

type ImageInfo struct {
	URL      string
	Filename string
}

func ParseCID(cid string) (id1, id2 int64, ok bool) {
	parts := strings.Split(cid, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}

	id1, err1 := strconv.ParseInt(parts[0], 10, 64)
	id2, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return id1, id2, true
}

func GetOtherUserID(cid string, currentUserID int64) (int64, error) {
	id1, id2, ok := ParseCID(cid)
	if !ok {
		return 0, fmt.Errorf("invalid CID format: %s", cid)
	}

	if id1 == currentUserID {
		return id2, nil
	}
	return id1, nil
}

func findMostFrequentID(idCounts map[int64]int) int64 {
	var mostFrequentID int64
	maxCount := 0
	for id, count := range idCounts {
		if count > maxCount {
			maxCount = count
			mostFrequentID = id
		}
	}
	return mostFrequentID
}

func extractContentText(contentType MessageContentType, contentJson string) string {
	var content map[string]any
	if err := json.Unmarshal([]byte(contentJson), &content); err != nil {
		return ""
	}

	switch contentType {
	case MessageContentTypeText:
		if text, ok := content["text"].(string); ok {
			return text
		}
	case MessageContentTypeImage:
		if filename, ok := content["filename"].(string); ok {
			return "[图片] " + filename
		}
		logger.Warn("Image content fallback to default, contentType=%d", contentType)
		return "[图片]"
	case MessageContentTypeDocument:
		if filename, ok := content["filename"].(string); ok {
			return "[文件] " + filename
		}
		return "[文件]"
	case MessageContentTypeShareLink:
		return extractAttachmentUrl(content, "[链接]")
	case MessageContentTypeLink:
		return extractFromAttachments(content, "b_tl", "[链接]", contentType)
	case MessageContentTypeFileOld, MessageContentTypeFile:
		return extractFromAttachments(content, "f_name", "[文件]", contentType)
	case MessageContentTypeFolder:
		return extractFromAttachments(content, "f_name", "[文件夹]", contentType)
	case MessageContentTypeSticker:
		return "[表情]"
	case MessageContentTypeCard:
		return "[名片]"
	case MessageContentTypeVideo:
		return extractFromAttachments(content, "title", "[视频]", contentType)
	case MessageContentTypeShortVideo:
		return "[短视频]"
	case MessageContentTypeVideoCall:
		return extractFromAttachments(content, "title", "[视频通话]", contentType)
	case MessageContentTypeCalendar:
		return "[日程]"
	case MessageContentTypeVote:
		return "[投票]"
	case MessageContentTypeRobot:
		return extractI18nMessage(content, "interactiveCardLastMessage", "LastMessageI18n", "zh_CN", "[群公告]", contentType)
	case MessageContentTypeActionCard:
		return extractI18nMessage(content, "interactiveCardLastMessage", "LastMessageI18n", "zh_CN", "[互动卡片]", contentType)
	case MessageContentTypeMiniProgram:
		return extractFromAttachments(content, "desc", "[小程序]", contentType)
	}
	return ""
}

func extractAttachmentUrl(content map[string]any, fallback string) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		return fallback
	}
	att, ok := attachments[0].(map[string]any)
	if !ok {
		return fallback
	}
	if url, ok := att["url"].(string); ok && url != "" {
		return url
	}
	return fallback
}

func extractFromAttachments(content map[string]any, field, fallback string, contentType MessageContentType) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		logger.Warn("Content fallback to default, contentType=%d, reason=no attachments", contentType)
		return fallback
	}

	att, ok := attachments[0].(map[string]any)
	if !ok {
		logger.Warn("Content fallback to default, contentType=%d, reason=invalid attachment", contentType)
		return fallback
	}

	extStr, ok := att["extension"].(string)
	if !ok {
		logger.Warn("Content fallback to default, contentType=%d, reason=no extension", contentType)
		return fallback
	}

	var ext map[string]any
	if err := json.Unmarshal([]byte(extStr), &ext); err != nil {
		logger.Warn("Content fallback to default, contentType=%d, reason=invalid extension json", contentType)
		return fallback
	}

	if val, ok := ext[field].(string); ok && val != "" {
		return val
	}

	logger.Warn("Content fallback to default, contentType=%d, reason=field '%s' not found", contentType, field)
	return fallback
}

func extractI18nMessage(content map[string]any, primaryField, i18nField, locale, fallback string, contentType MessageContentType) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		logger.Warn("Content fallback to default, contentType=%d, reason=no attachments", contentType)
		return fallback
	}

	att, ok := attachments[0].(map[string]any)
	if !ok {
		logger.Warn("Content fallback to default, contentType=%d, reason=invalid attachment", contentType)
		return fallback
	}

	extStr, ok := att["extension"].(string)
	if !ok {
		logger.Warn("Content fallback to default, contentType=%d, reason=no extension", contentType)
		return fallback
	}

	var ext map[string]any
	if err := json.Unmarshal([]byte(extStr), &ext); err != nil {
		logger.Warn("Content fallback to default, contentType=%d, reason=invalid extension json", contentType)
		return fallback
	}

	if val, ok := ext[primaryField].(string); ok && val != "" {
		return val
	}

	i18nStr, ok := ext[i18nField].(string)
	if !ok {
		logger.Warn("Content fallback to default, contentType=%d, reason=no %s or %s", contentType, i18nField, primaryField)
		return fallback
	}

	var i18n map[string]any
	if err := json.Unmarshal([]byte(i18nStr), &i18n); err != nil {
		logger.Warn("Content fallback to default, contentType=%d, reason=invalid i18n json", contentType)
		return fallback
	}

	if val, ok := i18n[locale].(string); ok && val != "" {
		return val
	}
	logger.Warn("Content fallback to default, contentType=%d, reason=%s not found", contentType, locale)
	return fallback
}

func extractImageInfo(contentJson string) (ImageInfo, bool) {
	var content map[string]any
	if err := json.Unmarshal([]byte(contentJson), &content); err != nil {
		return ImageInfo{}, false
	}

	url, ok := content["url"].(string)
	if !ok || url == "" || !strings.Contains(url, "down.dingtalk.com") {
		return ImageInfo{}, false
	}

	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ImageInfo{}, false
	}
	filename := parts[len(parts)-1]

	return ImageInfo{URL: url, Filename: filename}, true
}
