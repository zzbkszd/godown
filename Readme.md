# Godown - 自由资源下载工具

本项目功能说起来很简单，就是支持下载数据到本地

特别鸣谢： [youtube-dl](https://github.com/ytdl-org/youtube-dl)

项目，为本项目提供了大量网站解析的参考

## 使用方法

正在开发阶段，若需调用，请参考godown_test.go文件

已开发完成的下载器可直接调用，需要科学上网的参考godown.shadownet包

需要自行配置LocalShadowConfig，或者通过shadow_pool获取免费的服务器

## 目前支持：

- bilibili av号视频下载
- pornhub 视频下载
- xvideos 视频下载
- 基于owllook.net的小说下载
- twitter 文本/图片/视频下载

## 基础设施：

- 直链shadowsocks服务器的网络客户端。无需本地启动shadowsocks代理
- 国内直连的shadowsocks免费服务器爬取
- m3u8视频下载器

## feature

- 尽量避免panic，提升系统可用性
- 提供基于web的控制台
- 提供资源查看
- 支持更多的视频下载，以国外优先
- 在此基础上支持番剧/电视剧整部下载
- 支持更多的小说下载
- 支持漫画下载
- 支持图片集下载
- 支持bt/ed2k资源
- 支持智能资源搜索
- 支持核心服务的分布式部署

## 核心服务

- shadowsocks为核心的代理池服务
- 资源下载服务 * n 
- web客户端服务
- 资源搜索服务

