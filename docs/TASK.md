# TASK-v1.1.md

# Phase 1

## Task 1

初始化项目结构

验收：

go build ./...

成功

---

## Task 2

实现配置加载模块

支持：

* app.yaml
* sources.yaml
* channels.yaml
* aliases.yaml
* quality.yaml

验收：

启动打印配置统计信息

---

## Task 3

定义数据模型

实现：

Channel

Config

Cache

结构体

---

# Phase 2

## Task 4

实现 IPTV 抓取器

要求：

* 并发下载
* 超时10秒
* 自动跳过失败源

验收：

打印抓取结果

---

## Task 5

实现 M3U 解析器

支持：

* EXTINF
* group-title

输出：

[]Channel

---

# Phase 3

## Task 6

实现频道标准化

读取：

aliases.yaml

生成：

Canonical

---

## Task 7

实现频道过滤

执行顺序：

exclude

↓

keep

验收：

仅保留目标频道

---

## Task 8

实现频道去重

输出：

map[string][]Channel

Key：

Canonical

---

# Phase 4

## Task 9

实现线路分析器

分析：

Resolution

Bitrate

Protocol

验收：

正确识别：

4K

1080P

720P

m3u8

flv

ts

---

## Task 10

实现测速器

支持：

HLS

FLV

TS

超时：

5秒

并发：

50 Worker

输出：

LatencyMs

---

## Task 11

实现 QualityScore

实现：

CalculateScore()

输入：

Resolution

Bitrate

Protocol

Latency

输出：

QualityScore

验收：

高画质线路得分高于低画质线路

---

# Phase 5

## Task 12

实现缓存模块

目录：

/cache

支持：

读取缓存

写入缓存

TTL

默认：

24小时

---

## Task 13

实现最佳线路选择器

规则：

选择 QualityScore 最高线路

输出：

[]Channel

每频道仅保留一条线路

---

# Phase 6

## Task 14

实现 M3U 生成器

输出：

/output/final.m3u

分组：

CCTV

重庆

卫视

---

# Phase 7

## Task 15

实现 Builder Pipeline

流程：

Load Config

↓

Fetch Sources

↓

Parse M3U

↓

Normalize

↓

Filter

↓

Dedupe

↓

Analyze Quality

↓

Speed Test

↓

Calculate Score

↓

Select Best

↓

Generate M3U

↓

Exit

---

# Phase 8

## Task 16

Docker 化

实现：

Dockerfile

多阶段构建

Alpine Runtime

---

## Task 17

运行脚本

支持：

docker run --rm 
-v ./config:/config 
-v ./output:/output 
-v ./cache:/cache 
iptv-builder

---

# 最终验收

执行：

docker run --rm 
-v ./config:/config 
-v ./output:/output 
-v ./cache:/cache 
iptv-builder

必须满足：

1. 成功抓取 IPTV 源
2. 成功解析频道
3. 成功过滤频道
4. 成功分析线路质量
5. 成功测速
6. 成功计算评分
7. 成功选择最佳线路
8. 成功生成 final.m3u
9. 容器自动退出

性能目标：

频道数：

30~50

线路数：

200~1000

构建时间：

≤10分钟

内存：

≤256MB

CPU：

≤1 Core
