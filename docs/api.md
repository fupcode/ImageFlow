# ImageFlow API 参考

本文档包含 ImageFlow 的完整 API 说明。默认服务地址为：

```text
http://localhost:8686
```

## 公开端点

### 获取随机图片

```http
GET /api/random
```

查询参数：

| 参数 | 类型 | 描述 |
|------|------|------|
| `tag` | string | 按单个标签过滤 |
| `tags` | string | 按多个标签过滤（逗号分隔，AND 逻辑） |
| `exclude` | string | 排除包含这些标签的图片（逗号分隔） |
| `orientation` | string | 强制方向：`portrait` 或 `landscape` |
| `format` | string | 首选格式：`avif`、`webp` 或 `original` |

示例：

```bash
# 基础随机图片
curl "http://localhost:8686/api/random"

# 按标签过滤
curl "http://localhost:8686/api/random?tags=nature,landscape"

# 排除敏感内容
curl "http://localhost:8686/api/random?tag=wallpaper&exclude=nsfw,private"

# 强制竖屏方向
curl "http://localhost:8686/api/random?orientation=portrait&format=webp"
```

### 获取配置

```http
GET /api/config
```

返回前端可安全读取的运行配置：

```json
{
  "maxUploadCount": 20,
  "imageQuality": 80,
  "speed": 5,
  "avifSupport": false
}
```

| 字段 | 类型 | 描述 |
|------|------|------|
| `maxUploadCount` | number | 单次上传最大图片数 |
| `imageQuality` | number | 图片转换质量（1-100） |
| `speed` | number | 编码速度（0=最慢/最佳，8=最快） |
| `avifSupport` | boolean | 是否启用 AVIF 格式支持 |

## 认证端点

所有管理端点需要 `Authorization` 请求头：

```http
Authorization: Bearer your-api-key
```

### 上传图片

```http
POST /api/upload
Content-Type: multipart/form-data
```

| 字段 | 类型 | 描述 |
|------|------|------|
| `images[]` | file | 要上传的图片文件（支持多个） |
| `tags` | string | 逗号分隔的标签 |
| `expiryMinutes` | number | N 分钟后自动删除（可选） |

成功响应：

```json
{
  "results": [
    {
      "filename": "example.jpg",
      "status": "success",
      "message": "File uploaded and converted successfully",
      "orientation": "landscape",
      "format": "jpeg",
      "urls": {
        "original": "/images/original/landscape/20260520_120000_1234.jpg",
        "webp": "/images/landscape/webp/20260520_120000_1234.webp",
        "thumb": "/images/landscape/thumb/20260520_120000_1234.webp",
        "avif": "/images/landscape/avif/20260520_120000_1234.avif"
      },
      "expiryTime": "2026-05-20T13:00:00+08:00",
      "tags": ["nature", "wallpaper"]
    }
  ]
}
```

`results` 数组中的每一项对应一个上传文件：

| 字段 | 类型 | 描述 |
|------|------|------|
| `filename` | string | 原始上传文件名 |
| `status` | string | 单个文件处理结果：`success` 或 `error` |
| `message` | string | 处理结果说明；失败时包含错误原因 |
| `orientation` | string | 图片方向：`landscape` 或 `portrait`，仅成功时返回 |
| `format` | string | 检测到的原始格式，如 `jpeg`、`png`、`gif`、`webp`、`avif`，仅成功时返回 |
| `urls` | object | 可访问的图片 URL，包含 `original`、`webp`；普通图片包含 `thumb` 640px WebP 缩略图；启用 AVIF 时包含 `avif`。GIF 或转换失败时，`webp`/`avif` 可能回退为原图 URL，GIF 不生成 `thumb` |
| `expiryTime` | string | RFC3339 格式的过期时间；未设置 `expiryMinutes` 或值为 `0` 时为空字符串 |
| `tags` | string[] | 上传时传入并清理后的标签列表 |

如果单个文件处理失败，接口仍会返回 `200 OK`，并在对应结果项中标记失败：

```json
{
  "results": [
    {
      "filename": "broken.txt",
      "status": "error",
      "message": "Error reading image configuration: image: unknown format"
    }
  ]
}
```

认证失败或请求参数错误会返回统一错误格式：

```json
{
  "code": 1001,
  "message": "未上传文件"
}
```

### 列出图片

```http
GET /api/images
```

| 参数 | 类型 | 描述 |
|------|------|------|
| `page` | number | 页码 |
| `format` | string | 图片类型筛选：`all`、`image` 或 `gif` |
| `tag` | string | 按标签过滤 |
| `orientation` | string | 按方向过滤 |

### 删除图片

```http
POST /api/delete-image
Content-Type: application/json

{"id": "image-uuid"}
```

### 更新图片标签

```http
POST /api/update-tags
Content-Type: application/json

{
  "id": "image-uuid",
  "tags": ["nature", "wallpaper"]
}
```

`tags` 会自动去除前后空格、过滤空标签并去重。传入空数组可以清空该图片的所有标签。

成功响应：

```json
{
  "success": true,
  "message": "Tags updated successfully",
  "id": "image-uuid",
  "tags": ["nature", "wallpaper"]
}
```

### 获取所有标签

```http
GET /api/tags
```
