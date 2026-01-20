package downloader

import (
	"dingtalk/internal/logger"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DownloadTask struct {
	URL      string
	SavePath string
}

func downloadImage(url, savePath, token string) error {
	if _, err := os.Stat(savePath); err == nil {
		logger.Debug("Skipped existing: %s", filepath.Base(savePath))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", fmt.Sprintf("account=%s; deviceid=1;", token))
	req.Header.Set("Referer", "https://im.dingtalk.com/")
	req.Header.Set("Origin", "https://im.dingtalk.com")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Downloaded: %s", filepath.Base(savePath))
	return nil
}

func DownloadImagesParallel(tasks []DownloadTask, token string, workers int) {
	if len(tasks) == 0 {
		return
	}

	taskChan := make(chan DownloadTask, len(tasks))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				if err := downloadImage(task.URL, task.SavePath, token); err != nil {
					logger.Error("Failed to download %s: %v", task.URL, err)
				}
			}
		}()
	}

	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	wg.Wait()
}
