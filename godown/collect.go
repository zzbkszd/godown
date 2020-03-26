package godown

import "github.com/zzbkszd/godown/godown/downloader"

/**
数据集和下载任务
*/
var TYPE_VIDEO = 1

type DownloadTask struct {
	Dist       string
	Source     string
	Type       int
	Downloader downloader.Downloader
}

type Collect struct {
	Name        string   // 集合名称
	Type        int      // 集合类型
	Description string   // 集合描述
	Cover       string   // 集合封面（可选）
	Source      []string // 集合下载列表
}
