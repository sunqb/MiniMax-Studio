# MiniMax Studio

基于 MiniMax API 的多功能 AI 创作平台，支持文本生成、语音合成、音乐生成、声音复刻，生成内容可一键上传至 Cloudflare R2。前后端一体，单镜像部署。

## 功能

- **文本生成**：调用 MiniMax Chat API（OpenAI 兼容接口），支持系统提示词、温度调节、模型切换
- **语音合成 (TTS)**：20+ 内置音色，支持语速/音量/音调调节，输出 MP3/WAV/FLAC
- **音乐生成**：支持风格描述、歌词输入、纯音乐模式、AI 自动生成歌词
- **声音复刻**：上传 10s–5min 音频，克隆专属音色，可选试听
- **文件管理**：列出已上传到 MiniMax 的文件，支持按 purpose 筛选和删除
- **Cloudflare R2 上传**：生成内容一键上传，返回公开访问 URL

## Token Plan 模型对照

| 功能 | Token Plan 可用模型 | 按量付费模型 |
|------|-------------------|-------------|
| 文本生成 | MiniMax-M2.7 / M2.7-highspeed | 同左 + 其他系列 |
| 语音合成 | speech-2.8-hd / speech-2.8-turbo（Plus 套餐及以上） | speech-01/02/2.6/2.8 全系列 |
| 音乐生成 | music-2.6（所有套餐，100首/天限免） | music-2.6-free |

> Token Plan Key 与按量付费 API Key 相互独立，请确认使用对应的 Key。

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入你的配置
```

```env
# MiniMax（必填）
MINIMAX_API_KEY=your_token_plan_key_or_api_key
MINIMAX_BASE_URL=https://api.minimaxi.com   # 可选，默认此值

# Cloudflare R2（必填）
R2_ACCOUNT_ID=your_cloudflare_account_id
R2_ACCESS_KEY_ID=your_r2_access_key_id
R2_SECRET_ACCESS_KEY=your_r2_secret_access_key
R2_BUCKET_NAME=your_bucket_name
R2_PUBLIC_URL=https://pub-xxx.r2.dev        # R2 公开访问域名

# 服务器（可选）
PORT=8080
GIN_MODE=release
```

### 2. Docker 运行（推荐）

```bash
docker-compose up -d

# 查看日志
docker-compose logs -f
```

访问 http://localhost:8080

### 3. 本地开发

```bash
go run ./cmd/server
```

### 4. 手动构建

```bash
make docker-build
make docker-run
```

## 项目结构

```
minimax-studio/
├── cmd/server/main.go          # 程序入口
├── internal/
│   ├── config/config.go        # 配置加载（.env）
│   ├── minimax/
│   │   ├── client.go           # HTTP 基础客户端（连接池）
│   │   ├── text.go             # 文本生成
│   │   ├── tts.go              # 语音合成
│   │   ├── music.go            # 音乐生成
│   │   ├── clone.go            # 声音复刻（文件上传 + voice_clone）
│   │   └── files.go            # 文件列表 & 删除
│   ├── storage/r2.go           # Cloudflare R2 存储
│   ├── handler/                # Gin HTTP 路由 & 处理器
│   └── webfs/                  # 前端静态文件（go:embed）
├── Dockerfile                  # 两阶段构建
├── docker-compose.yml
└── .env.example
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/text/generate` | 文本生成 |
| POST | `/api/tts/synthesize` | 语音合成 |
| GET  | `/api/tts/voices` | 获取内置音色列表 |
| POST | `/api/music/generate` | 音乐生成（异步，立即返回 task_id） |
| GET  | `/api/music/task/:id` | 查询音乐生成任务状态 |
| GET  | `/api/music/models` | 获取音乐模型列表 |
| POST | `/api/voice/clone` | 声音复刻（multipart/form-data） |
| GET  | `/api/files` | 列出 MiniMax 已上传文件 |
| POST | `/api/files/delete` | 删除 MiniMax 文件 |
| POST | `/api/upload` | 上传文件到 R2 |
| GET  | `/health` | 健康检查 |

---

## cURL 示例

### 文本生成

```bash
curl -X POST http://localhost:8080/api/text/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "system_prompt": "你是一个专业的写作助手。",
    "user_message": "写一首关于秋天的短诗",
    "max_tokens": 1024,
    "temperature": 0.7
  }'
```

**响应：**
```json
{
  "content": "秋风拂过金黄的叶...",
  "model": "MiniMax-M2.7",
  "prompt_tokens": 32,
  "total_tokens": 128
}
```

---

### 语音合成（Token Plan 使用 speech-2.8-hd）

```bash
curl -X POST http://localhost:8080/api/tts/synthesize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "你好，欢迎使用 MiniMax Studio。",
    "model": "speech-2.8-hd",
    "voice_id": "female-shaonv",
    "speed": 1.0,
    "vol": 1.0,
    "pitch": 0,
    "format": "mp3"
  }'
```

**响应：**
```json
{
  "audio": "<base64编码的MP3数据>",
  "format": "mp3",
  "size": 59321
}
```

**可用音色（voice_id）：**

| voice_id | 说明 |
|----------|------|
| male-qn-qingse | 青涩青年 |
| male-qn-jingying | 精英青年 |
| male-qn-badao | 霸道青年 |
| female-shaonv | 少女 |
| female-yujie | 御姐 |
| female-chengshu | 成熟女性 |
| presenter_male | 男性主持人 |
| presenter_female | 女性主持人 |
| audiobook_male_1 | 男性有声书 |
| audiobook_female_1 | 女性有声书 |

---

### 音乐生成（带歌词）

```bash
# 1. 提交生成任务
curl -X POST http://localhost:8080/api/music/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "流行音乐, 轻快, 阳光",
    "lyrics": "[Verse]\n阳光照进窗台\n微风拂过脸颊\n[Chorus]\n今天是美好的一天",
    "is_instrumental": false,
    "lyrics_optimizer": false,
    "format": "mp3"
  }'
```

**提交响应：**
```json
{
  "task_id": "a1b2c3d4e5f6",
  "status": "pending"
}
```

### 音乐生成（纯音乐，无人声）

```bash
curl -X POST http://localhost:8080/api/music/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "古典钢琴, 宁静, 适合冥想",
    "is_instrumental": true,
    "format": "mp3"
  }'
```

### 音乐生成（AI 自动生成歌词）

```bash
curl -X POST http://localhost:8080/api/music/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "民谣, 思念, 秋天",
    "lyrics_optimizer": true,
    "format": "mp3"
  }'
```

### 查询音乐生成任务状态

```bash
# 2. 轮询任务状态（每 3 秒查询一次）
curl http://localhost:8080/api/music/task/a1b2c3d4e5f6
```

**进行中响应：**
```json
{
  "id": "a1b2c3d4e5f6",
  "type": "music",
  "status": "running",
  "created_at": 1716360000000
}
```

**完成响应：**
```json
{
  "id": "a1b2c3d4e5f6",
  "type": "music",
  "status": "completed",
  "created_at": 1716360000000,
  "result": {
    "audio": "<base64编码的MP3数据>",
    "format": "mp3",
    "size": 2977587,
    "history_id": "a1b2c3d4e5f6"
  }
}
```

**失败响应：**
```json
{
  "id": "a1b2c3d4e5f6",
  "type": "music",
  "status": "failed",
  "created_at": 1716360000000,
  "error": "music API error 1001: ..."
}
```

---

### 声音复刻

```bash
# 上传音频文件并克隆音色（不含试听）
curl -X POST http://localhost:8080/api/voice/clone \
  -F "file=@/path/to/sample.mp3" \
  -F "voice_id=my_custom_voice_01" \
  -F "noise_reduction=true" \
  -F "volume_normalization=true"
```

```bash
# 克隆音色并生成试听（需要 Plus+ Token Plan）
curl -X POST http://localhost:8080/api/voice/clone \
  -F "file=@/path/to/sample.mp3" \
  -F "voice_id=my_custom_voice_01" \
  -F "preview_text=你好，这是我的专属声音。" \
  -F "noise_reduction=true"
```

**响应：**
```json
{
  "voice_id": "my_custom_voice_01",
  "demo_audio": "https://cdn.minimaxi.com/audio/demo_xxxx.mp3"
}
```

> **说明**：
> - `file`：音频文件（mp3/wav/m4a），时长 10s–5min，大小 < 20MB
> - `voice_id`：自定义音色 ID，英文字母开头，8–256 字符
> - `preview_text`：填写后触发试听生成，返回 `demo_audio` URL（需 Plus+ 套餐）
> - 复刻后的 `voice_id` 可直接用于 `/api/tts/synthesize` 的 `voice_id` 字段

---

### 文件管理

```bash
# 列出所有上传文件
curl http://localhost:8080/api/files

# 按用途过滤（voice_clone / prompt_audio）
curl "http://localhost:8080/api/files?purpose=voice_clone"
```

**响应：**
```json
{
  "files": [
    {
      "file_id": 123456789,
      "filename": "sample.mp3",
      "bytes": 524288,
      "purpose": "voice_clone",
      "created_at": 1716000000
    }
  ]
}
```

```bash
# 删除文件
curl -X POST http://localhost:8080/api/files/delete \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": 123456789,
    "purpose": "voice_clone"
  }'
```

**响应：**
```json
{
  "deleted": true,
  "file_id": 123456789
}
```

---

### 上传到 Cloudflare R2

```bash
# 上传二进制文件
curl -X POST http://localhost:8080/api/upload \
  -F "file=@/path/to/audio.mp3" \
  -F "content_type=audio/mpeg"
```

**响应：**
```json
{
  "url": "https://pub-xxx.r2.dev/audio/a1b2c3d4e5f6.mp3",
  "key": "audio/a1b2c3d4e5f6.mp3",
  "size": 2977587
}
```

---

### 获取音色列表

```bash
curl http://localhost:8080/api/tts/voices
```

### 获取音乐模型列表

```bash
curl http://localhost:8080/api/music/models
```

### 健康检查

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## MiniMax Token Plan 原始 API 调用记录

> Token Plan Key 通过标准 Bearer Auth 调用，与按量付费 API Key 格式完全相同，只是模型名称和配额规则不同。
> Base URL：`https://api.minimaxi.com`（备用北京节点：`https://api-bj.minimaxi.com`）

### 文本生成 — POST /v1/chat/completions

OpenAI 兼容接口，Token Plan 支持 `MiniMax-M2.7` / `MiniMax-M2.7-highspeed`（极速版套餐）。

```bash
curl -X POST https://api.minimaxi.com/v1/chat/completions \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [
      {"role": "system", "content": "你是一个专业的写作助手。"},
      {"role": "user",   "content": "写一首关于秋天的短诗"}
    ],
    "max_tokens": 1024,
    "temperature": 0.7,
    "stream": false
  }'
```

**响应结构：**
```json
{
  "id": "...",
  "choices": [
    {
      "message": {"role": "assistant", "content": "..."},
      "finish_reason": "stop",
      "index": 0
    }
  ],
  "usage": {"prompt_tokens": 32, "completion_tokens": 96, "total_tokens": 128},
  "model": "MiniMax-M2.7",
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

---

### 语音合成 — POST /v1/t2a_v2

Token Plan Plus 及以上支持 `speech-2.8-hd` / `speech-2.8-turbo`。
音频数据以 **hex 编码**返回（非 base64）。

```bash
curl -X POST https://api.minimaxi.com/v1/t2a_v2 \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，欢迎使用 MiniMax Studio。",
    "stream": false,
    "voice_setting": {
      "voice_id": "female-shaonv",
      "speed": 1.0,
      "vol": 1.0,
      "pitch": 0
    },
    "audio_setting": {
      "sample_rate": 32000,
      "bitrate": 128000,
      "format": "mp3",
      "channel": 1
    }
  }'
```

**响应结构：**
```json
{
  "data": {
    "audio": "<hex编码的音频二进制>",
    "status": 2
  },
  "extra_info": {
    "audio_length": 3541,
    "audio_sample_rate": 32000,
    "audio_size": 59321,
    "usage_characters": 15
  },
  "trace_id": "...",
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

> **注意**：`data.audio` 是 hex 字符串，需 `hex.DecodeString()` 解码，不是 base64。

---

### 音乐生成 — POST /v1/music_generation

Token Plan 所有套餐支持 `music-2.6`，每日 100 首（限免）。
`music-2.6-free` 仅限按量付费用户，Token Plan 不可用。
音频数据优先 hex 编码，部分情况返回 base64。

```bash
# 纯音乐（无人声）
curl -X POST https://api.minimaxi.com/v1/music_generation \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "流行音乐, 轻快, 阳光",
    "is_instrumental": true,
    "stream": false,
    "audio_setting": {
      "sample_rate": 44100,
      "bitrate": 256000,
      "format": "mp3"
    }
  }'

# 带歌词
curl -X POST https://api.minimaxi.com/v1/music_generation \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "民谣, 思念, 秋天",
    "lyrics": "[Verse]\n落叶飘零秋风起\n[Chorus]\n思念如风无处寻",
    "is_instrumental": false,
    "lyrics_optimizer": false,
    "stream": false,
    "audio_setting": {
      "sample_rate": 44100,
      "bitrate": 256000,
      "format": "mp3"
    }
  }'
```

**响应结构：**
```json
{
  "data": {
    "audio": "<hex或base64编码的音频二进制>",
    "status": 2
  },
  "trace_id": "...",
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

> **注意**：
> - 有歌词时 `lyrics` 必填，`is_instrumental: true` 时可省略 `lyrics`
> - `lyrics_optimizer: true` 时 API 根据 `prompt` 自动生成歌词，无需手动填写
> - 生成耗时约 30–90 秒，建议客户端超时设置 ≥ 180 秒

---

### 声音复刻 — POST /v1/files/upload + POST /v1/voice_clone

声音复刻分两步：先上传音频文件获取 `file_id`，再调用复刻接口。

**第一步：上传音频文件**

```bash
curl -X POST https://api.minimaxi.com/v1/files/upload \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -F "purpose=voice_clone" \
  -F "file=@/path/to/sample.mp3"
```

**响应：**
```json
{
  "file": {
    "file_id": 123456789,
    "bytes": 524288,
    "filename": "sample.mp3",
    "purpose": "voice_clone",
    "created_at": 1716000000
  },
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

**第二步：克隆音色**

```bash
curl -X POST https://api.minimaxi.com/v1/voice_clone \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": 123456789,
    "voice_id": "my_custom_voice_01",
    "need_noise_reduction": true,
    "need_volume_normalization": true
  }'
```

**带试听（需要 Plus+ 套餐）：**

```bash
curl -X POST https://api.minimaxi.com/v1/voice_clone \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": 123456789,
    "voice_id": "my_custom_voice_01",
    "clone_prompt": {
      "text": "你好，这是我的专属声音。",
      "model": "speech-2.8-hd"
    },
    "need_noise_reduction": true
  }'
```

**响应：**
```json
{
  "demo_audio": "https://cdn.minimaxi.com/audio/demo_xxxx.mp3",
  "input_sensitive": false,
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

> 复刻成功的 `voice_id` 可直接传入 `/v1/t2a_v2` 的 `voice_setting.voice_id` 使用。

---

### 文件管理 — GET /v1/files/list + POST /v1/files/delete

**列出文件：**

```bash
# 列出所有文件
curl "https://api.minimaxi.com/v1/files/list" \
  -H "Authorization: Bearer $MINIMAX_API_KEY"

# 按用途过滤（voice_clone / prompt_audio / t2a_async_input）
curl "https://api.minimaxi.com/v1/files/list?purpose=voice_clone" \
  -H "Authorization: Bearer $MINIMAX_API_KEY"
```

**响应：**
```json
{
  "files": [
    {
      "file_id": 123456789,
      "bytes": 524288,
      "filename": "sample.mp3",
      "purpose": "voice_clone",
      "created_at": 1716000000
    }
  ],
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

**删除文件（注意：是 POST 请求，非 HTTP DELETE）：**

```bash
curl -X POST https://api.minimaxi.com/v1/files/delete \
  -H "Authorization: Bearer $MINIMAX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": 123456789,
    "purpose": "voice_clone"
  }'
```

**响应：**
```json
{
  "file_id": 123456789,
  "base_resp": {"status_code": 0, "status_msg": "success"}
}
```

> **注意**：MiniMax 文件删除 API 使用 `POST /v1/files/delete`（JSON body），而非标准 REST 的 `DELETE /v1/files/{id}`。

---

## MiniMax 平台

- 官网：[platform.minimaxi.com](https://platform.minimaxi.com)
- Token Plan 详情：[platform.minimaxi.com/docs/token-plan/intro](https://platform.minimaxi.com/docs/token-plan/intro)
- API 文档：[platform.minimaxi.com/docs/api-reference](https://platform.minimaxi.com/docs/api-reference)

---

## TODO

- [ ] TTS 语音合成异步任务化（通常 <10s，低优先级）
- [ ] 图像生成异步任务化
- [ ] 文本生成停止按钮 + Markdown 渲染
- [ ] 历史搜索 + 详情接口
- [ ] 一键成歌应用
- [ ] TTS 音色试听 + 批量配音
- [ ] 图像自动保存 R2
- [ ] 简单鉴权/访问密码
