# iptv-builder

IPTV 自动构建工具，运行于飞牛 NAS Docker 环境。

自动抓取多个 IPTV 源，解析频道，分析线路质量，测速评分，生成可直接供飞牛影视订阅的 `final.m3u`。

## 特性

- **多源抓取** — 并发下载多个 IPTV M3U 源，自动跳过失败源
- **频道标准化** — 别名映射统一频道名称
- **频道过滤** — 保留目标频道，剔除购物/广告/测试等
- **线路质量分析** — 识别分辨率 (4K/1080P/720P)、码率 (BANDWIDTH)、协议类型
- **测速** — 支持 HLS/FLV/TS，50 并发 worker，5 秒超时
- **综合评分** — 分辨率 + 码率 + 协议 + 延迟 四维加权，选最佳线路
- **缓存** — 测速结果缓存 24 小时，减少重复测速
- **一次运行即退出** — 非长期服务，由飞牛任务中心调度

## 工作原理

```
Load Config (5 个配置文件)
    ↓
Fetch Sources (并发 HTTP 抓取)
    ↓
Parse M3U (解析频道信息)
    ↓
Normalize (别名标准化)
    ↓
Filter (exclude → keep 过滤)
    ↓
Dedupe (按规范名去重)
    ↓
Analyze Quality (识别分辨率/码率/协议)
    ↓
Speed Test (HLS/FLV/TS 测速)
    ↓
Calculate Score (四维综合评分)
    ↓
Select Best (每频道选最高分线路)
    ↓
Generate final.m3u
    ↓
Exit
```

## 快速开始

### 准备配置文件

```bash
mkdir -p config output cache logs
```

**config/sources.yaml** — IPTV 源列表：

```yaml
sources:
  - https://example1.com/iptv.m3u
  - https://example2.com/iptv.m3u
```

**config/channels.yaml** — 频道过滤规则：

```yaml
keep:
  - CCTV
  - 重庆
  - 湖南卫视
  - 浙江卫视
  - 江苏卫视
  - 东方卫视
  - 北京卫视
  - 广东卫视
  - 深圳卫视

exclude:
  - 购物
  - 广告
  - 导视
  - 测试
  - 轮播
  - 广播
```

**config/aliases.yaml** — 频道名称标准化：

```yaml
央视综合: CCTV1
CCTV-1: CCTV1
CCTV1HD: CCTV1
湖南卫视HD: 湖南卫视
湖南卫视1080P: 湖南卫视
重庆卫视HD: 重庆卫视
```

**config/quality.yaml** — 评分权重：

```yaml
quality:
  resolution_weight: 0.5
  bitrate_weight: 0.2
  protocol_weight: 0.1
  latency_weight: 0.2
```

**config/app.yaml** — 运行参数：

```yaml
app:
  cache_ttl_hours: 24
  workers: 50
  log_level: info
```

### 构建镜像

```bash
docker build -t iptv-builder:latest .
```

### 运行

```bash
docker run --rm \
  -v $(pwd)/config:/config:ro \
  -v $(pwd)/output:/output \
  -v $(pwd)/cache:/cache \
  iptv-builder
```

运行成功后，`output/final.m3u` 即为可直接使用的直播源文件。

## 评分算法

每条线路的综合评分由四个维度加权计算：

| 维度 | 权重 | 说明 |
|------|------|------|
| 分辨率 | 50% | 4K(100分) > 1080P(70分) > 720P(40分) > SD(10分) |
| 码率 | 20% | ≥8Mbps(100分) > ≥4Mbps(70分) > ≥2Mbps(40分) |
| 协议 | 10% | HLS/m3u8(100分) > TS(50分) > FLV(30分) |
| 延迟 | 20% | ≤200ms(100分) > ≤500ms(70分) > ≤1s(40分) |

> 权重可通过 `config/quality.yaml` 自定义。

## 飞牛影视接入

在飞牛影视中添加直播源：

```
http://<NAS-IP>:8088/final.m3u
```

> 需要将 `output/` 目录通过 HTTP 服务暴露（如 Nginx、飞牛自带 Web 服务），将 `8088` 端口映射到 `/share/iptv/output/`。

## 飞牛定时任务

在飞牛任务中心创建定时任务：

```bash
docker run --rm \
  -v /share/iptv/config:/config:ro \
  -v /share/iptv/output:/output \
  -v /share/iptv/cache:/cache \
  iptv-builder
```

建议每天凌晨执行一次（如 `0 4 * * *`），缓存 TTL 24 小时与之匹配。

## 开发

### 环境要求

- Go 1.24+
- Docker

### 构建

```bash
make build          # 编译
make test           # 测试
make lint           # 代码检查
make run            # 本地运行（使用 configs/ 目录的示例配置）
```

### 目录结构

```
iptv-builder/
├── cmd/iptv-builder/     # 入口
├── internal/
│   ├── model/            # 数据模型
│   ├── config/           # 配置加载
│   ├── fetch/            # IPTV 源抓取
│   ├── parser/           # M3U 解析
│   ├── normalizer/       # 频道标准化
│   ├── filter/           # 频道过滤
│   ├── dedupe/           # 去重
│   ├── analyzer/         # 线路质量分析
│   ├── speedtest/        # 测速
│   ├── scorer/           # 综合评分
│   ├── cache/            # 缓存
│   ├── selector/         # 最佳线路选择
│   ├── generator/        # M3U 生成
│   └── builder/          # 流水线编排
├── configs/              # 示例配置
├── scripts/              # 运行脚本
├── Dockerfile
└── Makefile
```

## 性能

| 指标 | 目标 |
|------|------|
| 频道数 | 30 ~ 50 |
| 线路数 | 200 ~ 1000 |
| 构建时间 | ≤ 10 分钟 |
| 内存占用 | ≤ 256 MB |
| CPU 占用 | ≤ 1 Core |

## License

MIT — 详见 [LICENSE](LICENSE)
