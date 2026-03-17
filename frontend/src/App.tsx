import { FormEvent, useEffect, useState } from "react";

const emotionPreset = {
  id: "calm-bass-upright",
  name: "低音,正派,冷静",
  emotionPrompt: "https://cdn.kuse.ai/tutorials/reademo.wav",
};

type VoicePreset = {
  id: string;
  name: string;
  reference_audio: string;
};

type SynthesisJob = {
  id: string;
  status: "running" | "completed" | "failed";
  stage: string;
  message: string;
  progress: number;
  total_segments: number;
  completed_segments: number;
  error?: string;
  audio_url?: string;
};

const defaultText =
  "山色有无中，江流天地外。把一段长文字交给系统，它会按句子切开，逐句生成朗诵，再拼成完整音频。你可以选择一个固定参考音色，也可以只用固定情感预设直接生成。";

export default function App() {
  const [text, setText] = useState(defaultText);
  const [voicePresets, setVoicePresets] = useState<VoicePreset[]>([]);
  const [selectedVoicePresetId, setSelectedVoicePresetId] = useState("none");
  const [gapSeconds, setGapSeconds] = useState("0.1");
  const [audioUrl, setAudioUrl] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [segmentCount, setSegmentCount] = useState<number | null>(null);
  const [job, setJob] = useState<SynthesisJob | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const response = await fetch("/api/v1/voice-presets");
        if (!response.ok) {
          setError("加载参考音色预设失败");
          return;
        }

        const payload = (await response.json()) as { items: VoicePreset[] };
        setVoicePresets(payload.items);
        if (payload.items.some((item) => item.id === "boy")) {
          setSelectedVoicePresetId("boy");
          return;
        }
        if (payload.items[0]) {
          setSelectedVoicePresetId(payload.items[0].id);
        }
      } catch {
        setError("加载参考音色预设失败");
      }
    })();
  }, []);

  useEffect(() => {
    return () => {
      if (audioUrl.startsWith("blob:")) {
        URL.revokeObjectURL(audioUrl);
      }
    };
  }, [audioUrl]);

  const activeJobId = job?.status === "running" ? job.id : "";

  useEffect(() => {
    if (!activeJobId) {
      return;
    }

    let cancelled = false;
    const poll = async () => {
      try {
        const response = await fetch(`/api/v1/synthesize-jobs/${activeJobId}`);
        if (!response.ok) {
          throw new Error("进度查询失败");
        }

        const payload = (await response.json()) as SynthesisJob;
        if (cancelled) {
          return;
        }

        setJob(payload);
        if (payload.status === "completed") {
          setAudioUrl(payload.audio_url ?? `/api/v1/synthesize-jobs/${payload.id}/audio`);
          setSegmentCount(payload.total_segments);
          setLoading(false);
          return;
        }
        if (payload.status === "failed") {
          setError(payload.error ?? "生成失败");
          setLoading(false);
        }
      } catch (pollError) {
        if (!cancelled) {
          setError(pollError instanceof Error ? pollError.message : "进度查询失败");
          setLoading(false);
        }
      }
    };

    void poll();
    const timer = window.setInterval(() => {
      void poll();
    }, 800);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeJobId]);

  const paragraphCount = text
    .split("。")
    .map((item) => item.trim())
    .filter(Boolean).length;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    setSegmentCount(null);
    setJob(null);
    if (audioUrl.startsWith("blob:")) {
      URL.revokeObjectURL(audioUrl);
    }
    setAudioUrl("");

    try {
      const selectedVoicePreset = voicePresets.find((item) => item.id === selectedVoicePresetId);

      const response = await fetch("/api/v1/synthesize-jobs", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          text,
          reference_audio: selectedVoicePreset?.reference_audio ?? "",
          gap_seconds: gapSeconds.trim() === "" ? undefined : Number(gapSeconds),
          emotion_preset_id: emotionPreset.id,
        }),
      });

      if (!response.ok) {
        const payload = (await response.json()) as { error?: string };
        throw new Error(payload.error ?? "生成失败");
      }
      const payload = (await response.json()) as SynthesisJob;
      setJob(payload);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "生成失败");
      setLoading(false);
    }
  }

  const progressValue = job?.progress ?? 0;

  return (
    <main className="page-shell">
      <section className="hero-card">
        <p className="eyebrow">Read Article</p>
        <h1>把长文本直接变成朗诵音频</h1>
        <p className="hero-copy">
          后端会按句号切片，逐句调用远程 TTS，再把每段 WAV 拼成一条完整音频。
        </p>
      </section>

      <section className="workspace">
        <form className="composer" onSubmit={handleSubmit}>
          <label className="field">
            <span>长文本</span>
            <textarea
              value={text}
              onChange={(event) => setText(event.target.value)}
              rows={11}
              placeholder="请输入需要朗诵的长文本"
            />
          </label>

          <div className="field-grid">
            <label className="field">
              <span>参考音色</span>
              <select
                value={selectedVoicePresetId}
                onChange={(event) => setSelectedVoicePresetId(event.target.value)}
              >
                {voicePresets.map((preset) => (
                  <option key={preset.id} value={preset.id}>
                    {preset.name}
                  </option>
                ))}
              </select>
              <small className="field-note">
                参考音色改为固定预设列表。当前预设包含“正太”：
                <code>https://cdn.kuse.ai/tutorials/readvoice.wav</code>
              </small>
            </label>

            <label className="field">
              <span>句间停顿（秒）</span>
              <input
                value={gapSeconds}
                onChange={(event) => setGapSeconds(event.target.value)}
                inputMode="decimal"
                placeholder="0.1"
              />
            </label>
          </div>

          <div className="preset-card">
            <span className="preset-label">固定情感预设</span>
            <strong>{emotionPreset.name}</strong>
            <p>
              合成时会把 <code>emotion_prompt</code> 固定传为
              <code>{emotionPreset.emotionPrompt}</code>
            </p>
          </div>

          <div className="meta-row">
            <span>预估分段 {paragraphCount} 句</span>
            <span>参考音色与情感参考都走固定预设</span>
          </div>

          <button className="primary-button" disabled={loading || !text.trim()}>
            {loading ? "任务进行中..." : "生成朗诵音频"}
          </button>
        </form>

        <aside className="result-card">
          <p className="panel-title">生成结果</p>
          <p className="result-copy">
            参考音色来自固定预设；情感参考固定使用
            <code>{emotionPreset.name}</code>
            ，并把
            <code>{emotionPreset.emotionPrompt}</code>
            作为
            <code>emotion_prompt</code>
            传给推理服务。
          </p>

          {job ? (
            <div className="progress-card">
              <div className="progress-head">
                <strong>{job.message}</strong>
                <span>{Math.round(progressValue)}%</span>
              </div>
              <div className="progress-track" aria-hidden="true">
                <div className="progress-fill" style={{ width: `${progressValue}%` }} />
              </div>
              <p className="progress-copy">
                {job.stage === "splitting" ? "切片中" : null}
                {job.stage === "generating" ? `语音生成中 ${job.completed_segments}/${job.total_segments}` : null}
                {job.stage === "merging" ? "拼接中" : null}
                {job.stage === "completed" ? "已完成" : null}
                {job.stage === "failed" ? "生成失败" : null}
              </p>
            </div>
          ) : null}

          {error ? <p className="error-banner">{error}</p> : null}

          {audioUrl ? (
            <div className="audio-result">
              <audio controls src={audioUrl} />
              <a className="download-link" href={audioUrl} download="reading.wav">
                下载 reading.wav
              </a>
              {segmentCount !== null ? <p>本次共生成 {segmentCount} 个片段。</p> : null}
            </div>
          ) : (
            <div className="empty-state">
              <p>生成成功后，这里会出现播放器和下载链接。</p>
            </div>
          )}
        </aside>
      </section>
    </main>
  );
}
