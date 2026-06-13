# PRD-v1.1.md

# IPTV Builder

Version: 1.1

---

## 项目目标

构建一个运行于飞牛 NAS 的 IPTV 自动构建工具。

系统自动完成：

1. 抓取 IPTV 源
2. 解析 M3U
3. 标准化频道名称
4. 过滤目标频道
5. 去重
6. 分析线路质量
7. 测速
8. 计算综合评分
9. 选择最佳线路
10. 生成 final.m3u

最终供飞牛影视直接订阅。

---

# 核心设计原则

## 用户目标

优先获得：

* 更高清晰度
* 更高码率
* 更稳定协议
* 更低延迟

而不是单纯选择延迟最低线路。

---

# 部署模式

Docker Job

程序启动：

Load Config
→ Build
→ Output M3U
→ Exit

程序不常驻。

程序内部禁止：

* Cron
* Scheduler
* HTTP Server
* Web UI

调度由飞牛任务中心负责。

---

# 目录结构

/share/iptv

config/
output/
cache/
logs/

---

# 配置文件

config/

app.yaml
sources.yaml
channels.yaml
aliases.yaml
quality.yaml

---

# 频道范围

保留：

* CCTV*
* 重庆*
* 湖南卫视
* 浙江卫视
* 江苏卫视
* 东方卫视
* 北京卫视
* 广东卫视
* 深圳卫视

过滤：

* 购物
* 广告
* 导视
* 测试
* 轮播
* 广播

---

# 数据模型

```go
type Channel struct {
    Name string
    URL string
    Group string

    Canonical string

    Resolution string
    Bitrate int64
    Protocol string

    LatencyMs int64

    QualityScore float64

    Source string

    Valid bool
}
```

---

# 标准化

通过 aliases.yaml 完成。

示例：

央视综合 → CCTV1

CCTV-1 → CCTV1

湖南卫视HD → 湖南卫视

重庆卫视1080P → 重庆卫视

---

# 去重

按 Canonical 聚合。

示例：

CCTV1

线路A
线路B
线路C

↓

同一频道组

---

# 线路质量分析

## 分辨率识别

支持：

4K
2160P
UHD
HDR

1080P
1080I
FHD

720P

SD

来源：

1. 频道名称
2. HLS Master Playlist

---

## 码率识别

来源：

HLS Master Playlist

例如：

BANDWIDTH=8000000

---

## 协议识别

支持：

m3u8
flv
ts

---

# 测速

支持：

HLS
FLV
TS

超时：

5秒

测速方式：

下载最小可播放数据。

记录：

LatencyMs

---

# 综合评分

QualityScore

评分维度：

Resolution
Bitrate
Protocol
Latency

默认权重：

Resolution 50%

Bitrate 20%

Protocol 10%

Latency 20%

---

# 线路选择

每频道仅保留：

QualityScore 最高线路

不是延迟最低线路。

---

# 缓存

缓存目录：

/cache

缓存内容：

* Resolution
* Bitrate
* Protocol
* Latency
* QualityScore

默认有效期：

24小时

---

# 输出

生成：

/output/final.m3u

---

# 分组规则

频道名前缀：

CCTV

↓

CCTV

频道名前缀：

重庆

↓

重庆

其他：

↓

卫视

---

# 飞牛影视接入

直播源：

http://nas-ip:8088/final.m3u

---

# 非目标

禁止实现：

* Web后台
* API
* 用户系统
* SQLite
* MySQL
* Redis
* EPG管理
* 多线路保留
* 内部定时器
* 消息队列

---

# 技术栈

Go 1.24+

推荐：

cobra
viper
yaml.v3
ants

---

# 成功标准

执行：

docker run --rm iptv-builder

10分钟内完成构建。

生成：

/output/final.m3u

飞牛影视可正常导入并播放。
