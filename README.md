# NetFlow Flasher - 网络流量刷流工具

一个轻量级的网络下行流量消耗工具，用于测试带宽、刷流量或网络压力测试。

## 特性

- 🚀 **高效流式处理**：内存占用恒定，网络数据通过固定大小缓冲区传输到 `io.Discard`
- ⚡ **精确限速控制**：通过定时器控制下载速度，实现精确限速
- 🎯 **零磁盘IO**：所有数据直接丢弃，不存储到内存或磁盘
- 🔄 **循环下载**：支持多URL循环下载，模拟真实使用场景
- ⚙️ **灵活配置**：JSON配置文件，易于自定义

## 安装

```bash
# 直接运行（使用镜像内置默认配置）
docker run --rm hsux/netflowflasher

# 使用自定义配置
docker run --rm -v ./config.json:/app/config.json hsux/netflowflasher
```

## 配置文件

直接使用默认配置或自行修改 `config.json` 文件：

```json
{
  "downloadList": [
    "https://example.com/file1.zip",
    "https://example.com/file2.iso",
    "https://mirror.example.com/large-file.tar.gz"
  ],
  "datachunkMB": 5,
  "timelapse": 2
}
```

### 配置参数说明

| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `downloadList` | 数组 | 下载URL列表，程序会循环访问 | 见上方示例 |
| `datachunkMB` | 整数 | 每次下载的数据块大小（MB） | `5` = 每次5MB |
| `timelapse` | 整数 | 下载间隔时间（秒） | `2` = 每2秒下载一次 |

### 速度计算

实际下载速度 = `datachunkMB` ÷ `timelapse` MB/s

**配置示例：**

```json
// 配置1：2.5 MB/s 限速
{
  "datachunkMB": 5,
  "timelapse": 2
}

// 配置2：10 MB/s 限速  
{
  "datachunkMB": 10,
  "timelapse": 1
}

// 配置3：1 MB/s 限速
{
  "datachunkMB": 1,
  "timelapse": 1
}
```

## 使用场景

- **带宽测试**：测试网络连接的实际下载速度
- **流量消耗**：快速消耗数据流量包
- **网络压力测试**：模拟高并发下载场景
- **CDN性能测试**：测试不同CDN节点的响应性能
- **ISP限速检测**：检测运营商是否对特定服务限速

## 注意事项

⚠️ **重要提醒**
- 本工具会消耗网络流量，请注意流量费用
- 在计费网络（如移动数据）环境下使用需谨慎
- 建议在无限流量或内网环境下进行测试
- 请遵守相关网站的使用条款
