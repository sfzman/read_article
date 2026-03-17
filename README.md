# 朗诵生成器

一个最小的 Go + React 应用。用户输入长文本后，后端会按 `。` 分句，逐句调用远程 TTS 推理接口，最后把各段 WAV 音频按可调停顿拼接成完整朗诵。

## 功能

- 按 `。` 切分长文本并逐段合成
- 参考音色改为固定预设列表，当前内置 `"正太"`：`https://cdn.kuse.ai/tutorials/readvoice.wav`
- 固定情感预设 `"低音,正派,冷静"`，把 `https://cdn.kuse.ai/tutorials/reademo.wav` 作为 `emotion_prompt`
- 可调整句间停顿，默认 `0.1` 秒
- 前端显示任务进度：切片中、语音生成中 `(n/总数)`、拼接中
- React 前端支持试听和下载生成结果

## 后端启动

```bash
cp .env.example .env
cd backend
go run ./cmd/server
```

后端启动时会自动尝试读取：

- 项目根目录的 `.env`
- `backend/.env`

同名变量的优先级是：系统环境变量 > `backend/.env` > 根目录 `.env`。

可用环境变量：

- `SERVER_PORT`：默认 `18080`
- `INFERENCE_JWT_PRIVATE_KEY`：如果远程推理服务要求 JWT 鉴权，可以传 PEM 私钥
- `INFERENCE_JWT_EXPIRE_SECONDS`：JWT 过期秒数，默认 `300`
- `INFERENCE_TIMEOUT`：推理请求超时，默认 `5m`
- `DEFAULT_GAP_SECONDS`：默认停顿时长，默认 `0.1`

## 前端启动

```bash
cd frontend
npm install
npm run dev
```

Vite 已经把 `/api` 代理到 `http://localhost:18080`。

## 接口

`GET /api/v1/voice-presets`

返回固定参考音色预设列表，当前至少包含：

```json
{
  "items": [
    {
      "id": "none",
      "name": "不使用参考音色",
      "reference_audio": ""
    },
    {
      "id": "boy",
      "name": "正太",
      "reference_audio": "https://cdn.kuse.ai/tutorials/readvoice.wav"
    }
  ]
}
```

`POST /api/v1/synthesize-jobs`

创建一个异步朗诵任务，返回任务状态对象。

`GET /api/v1/synthesize-jobs/:id`

查询任务进度，返回示例：

```json
{
  "id": "job123",
  "status": "running",
  "stage": "generating",
  "message": "语音生成中 (3/20)",
  "progress": 22,
  "total_segments": 20,
  "completed_segments": 3
}
```

`GET /api/v1/synthesize-jobs/:id/audio`

任务完成后返回拼接好的 `audio/wav`。

`POST /api/v1/synthesize`

请求体：

```json
{
  "text": "第一句。第二句。",
  "reference_audio": "https://cdn.kuse.ai/tutorials/readvoice.wav",
  "gap_seconds": 0.1,
  "emotion_preset_id": "calm-bass-upright"
}
```

返回值为 `audio/wav` 二进制流。
