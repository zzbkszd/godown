# 数据扫描部分描述

## 功能概述：

输入：一个特定的Keys输入，用来定位一个资源/资源列表。
```
举例来说：
一个twitter用户名
一个视频列表页
一个小说列表页

也可以是指定的一个小说名关键字，进行检索并返回之类的操作。
```

scanner可以分为两种：

- 下载数据scanner， 每次下载时扫描并下载
- 订阅数据scanner， 就是定时扫描并获取基本信息，但是不下载的那种

输出：一个数据列表，是否下载根据配置选择特定的下载器进行下载

## 程序结构

### 扫描器核心 ScannerCore

扫描器是核心，用来根据输入得到输出。

扫描器应该是一个基准的爬虫核心程序。针对不同的数据源进行不同的衍生，参考downloader

不同的扫描器需要不同的入参，所以一次扫描请求必须是针对特定的扫描器构建的。。