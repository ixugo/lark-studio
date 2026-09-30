package conf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Bootstrap struct {
	Runtime  Runtime  // 运行时
	Server   Server   // 服务器
	Data     Data     // 数据
	Log      Log      // 日志
	Pipeline Pipeline // 流水线配置
	LLM      LLM      // LLM 翻译配置
	TTS      TTS      // TTS 配音配置
	LipSync  LipSync  // 对口型配置
}

// Pipeline 流水线处理配置
type Pipeline struct {
	Workers            int     `comment:"并发任务数（1~8，默认 2，保存后即时生效）" json:"workers"`
	WhisperMode        string  `comment:"ASR 引擎: whisper-cpp / openai" json:"whisper_mode"`
	WhisperBin         string  `comment:"whisper.cpp 可执行文件路径（留空优先使用应用内嵌运行时）" json:"whisper_bin"`
	WhisperModel       string  `comment:"whisper ggml 模型文件路径" json:"whisper_model"`
	ASRBaseURL         string  `comment:"OpenAI 兼容 ASR API 基础地址" json:"asr_base_url"`
	ASRAPIKey          string  `comment:"OpenAI 兼容 ASR API 密钥" json:"asr_api_key"`
	ASRModel           string  `comment:"OpenAI 兼容 ASR 模型名称" json:"asr_model"`
	FFmpegBin          string  `comment:"ffmpeg 路径（空则使用 PATH 中的）" json:"ffmpeg_bin"`
	DefaultOutputDir   string  `comment:"默认输出目录" json:"default_output_dir"`
	DefaultTargetLang  string  `comment:"默认目标语言" json:"default_target_lang"`
	TranslatePrompt    string  `comment:"自定义翻译系统提示词（留空使用内置默认）" json:"translate_prompt"`
	MaxSpeedFactor     float64 `comment:"TTS 调速上限（0 或 ≤1 不调速，推荐 1.2~1.3）" json:"max_speed_factor"`
	TranslateChunkSize int     `comment:"每次发给 LLM 的句子数（5~20，默认 10）" json:"translate_chunk_size"`
	TTSWorkers         int     `comment:"TTS 并发协程数（1~4，默认 2）" json:"tts_workers"`
	CleanIntermediate  bool    `comment:"处理完成后删除中间产物（raw.mp3/audio_segs 等）" json:"clean_intermediate"`
	SubtitleOutput     string  `comment:"字幕输出方式: soft(软字幕流复制秒级完成，推荐) / burn(烧录画面) / none(无字幕)" json:"subtitle_output"`
}

// LLM 大模型配置
type LLM struct {
	Provider  string `comment:"翻译服务: google / bing / deeplx / openai" json:"provider"`
	BaseURL   string `comment:"OpenAI 兼容 API 地址" json:"base_url"`
	APIKey    string `comment:"API 密钥" json:"api_key"`
	Model     string `comment:"模型名称" json:"model"`
	DeepLXURL string `comment:"DeepLX 自建接口地址" json:"deeplx_url"`
}

// TTS 语音合成配置
type TTS struct {
	Protocol     string `comment:"语音服务协议: openai / mlx" json:"protocol"`
	Language     string `comment:"MLX 合成语言，Auto 按任务目标语言" json:"language"`
	Instructions string `comment:"支持时使用的情绪和风格指令" json:"instructions"`
	EdgeVoice    string `comment:"Edge TTS 音色" json:"edge_voice"`
	OpenAIVoice  string `comment:"OpenAI 兼容 TTS 音色" json:"openai_voice"`
	Type         string `comment:"TTS 类型: edge / openai" json:"type"`
	Voice        string `comment:"默认语音" json:"voice"`
	BaseURL      string `comment:"OpenAI TTS API 地址（Type=openai 时生效）" json:"base_url"`
	APIKey       string `comment:"OpenAI TTS API 密钥" json:"api_key"`
	Model        string `comment:"OpenAI TTS 模型名" json:"model"`
}

// LipSync 对口型（唇形同步）配置
type LipSync struct {
	Enabled bool   `comment:"是否启用对口型（MuseTalk），仅 ModeDub 生效" json:"enabled"`
	BaseURL string `comment:"MuseTalk API 地址" json:"base_url"`
	APIKey  string `comment:"API 密钥（部分服务需要）" json:"api_key"`
}

type Runtime struct {
	Debug        bool   `toml:"-" json:"-"`
	BuildVersion string `toml:"-" json:"build_version"`
	ConfigDir    string `toml:"-" json:"config_dir"`
	ConfigPath   string `toml:"-" json:"config_path"`
}

type Server struct {
	HTTP ServerHTTP `comment:"对外提供的服务，建议由 nginx 代理"` // HTTP服务器
}

type ServerHTTP struct {
	Port      int         `comment:"http 端口"`                // 服务器端口号
	Timeout   Duration    `comment:"请求超时时间"`                 // 请求超时时间
	JwtSecret string      `comment:"jwt 秘钥，空串时，每次启动程序将随机赋值"` // JWT密钥
	PProf     ServerPPROF // Pprof配置
}

// ServerPPROF 结构体，包含 Enabled 和 AccessIps 两个字段
type ServerPPROF struct {
	Enabled   bool     `comment:"是否启用 pprof, 建议设置为 true"`  // 是否启用
	AccessIps []string `comment:"访问白名单" json:"access_ips"` // 允许访问的IP地址列表
}

// Data 结构体，包含 Database 和 Redis 两个字段
type Data struct {
	// Database 数据库
	Database Database `comment:"数据库支持 sqlite 和 postgres 两种，使用 sqlite 时 dsn 应当填写文件存储路径"`
	// Redis Redis数据库
	// Redis DataRedis
}

// Database 结构体，包含 Dsn、MaxIdleConns、MaxOpenConns、ConnMaxLifetime 和 SlowThreshold 五个字段
type Database struct {
	Dsn             string   // 数据源名称
	MaxIdleConns    int32    // 最大空闲连接数
	MaxOpenConns    int32    // 最大打开连接数
	ConnMaxLifetime Duration // 连接最大生命周期
	SlowThreshold   Duration // 慢查询阈值
}

// Log 结构体，包含 Dir、Level、MaxAge、RotationTime 和 RotationSize 五个字段
type Log struct {
	Name         string   `comment:"日志文件名(选填)"`
	Dir          string   `comment:"日志存储目录，不能使用特殊符号"`
	Level        string   `comment:"记录级别 debug/info/warn/error"`
	MaxAge       int      `comment:"保留日志多久，超过时间自动删除"`
	RotationTime Duration `comment:"多久时间，分割一个新的日志文件"`
	MaxSize      int      `comment:"多大文件，分割一个新的日志文件(MB)"`
	Compress     bool     `comment:"是否压缩日志"`
	MaxBackups   int      `comment:"保留的旧日志归档文件最大数量，超出的自动删除"`
}

type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
	x, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration().String()), nil
}

func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}

// DataDir 返回应用数据根目录 ~/dsub，自动创建
func DataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "dsub")
}

// StudioDir 返回 Lark Studio 集中资源管理根目录。
func StudioDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".lark-studio")
}

// DocumentsDir 返回用户系统的文档目录（macOS/Linux: ~/Documents，Windows: Documents）。
func DocumentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, "Documents")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// TasksDir 返回文档目录下的任务输出根目录。
func TasksDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Documents", "lark-studio")
}

// TaskOutputDir 将空值和旧版文档目录默认值归一为当前默认目录，保留自定义路径。
func TaskOutputDir(configured string) string {
	dir := strings.TrimSpace(configured)
	if dir == "" || filepath.Clean(dir) == filepath.Dir(TasksDir()) || filepath.Clean(dir) == filepath.Join(StudioDir(), "tasks") || dir == "~/Documents" {
		return TasksDir()
	}
	return dir
}

// EnsureDataDirs 确保数据目录结构存在
func EnsureDataDirs() error {
	base := DataDir()
	for _, sub := range []string{"configs", "backup", "logs"} {
		if err := os.MkdirAll(filepath.Join(base, sub), 0o755); err != nil {
			return err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	studioBase := StudioDir()
	if err := migrateStudioDir(filepath.Join(home, ".vdub_studio"), studioBase); err != nil {
		return err
	}
	for _, sub := range []string{"models", "runtime", "tasks"} {
		if err := os.MkdirAll(filepath.Join(studioBase, sub), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// migrateStudioDir 将旧资源目录迁入新位置，目标已有内容优先保留。
func migrateStudioDir(oldDir, newDir string) error {
	if _, err := os.Stat(oldDir); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("检查旧资源目录失败: %w", err)
	}
	if _, err := os.Stat(newDir); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(oldDir, newDir); err != nil {
			return fmt.Errorf("迁移旧资源目录失败: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("检查新资源目录失败: %w", err)
	}
	if err := mergeStudioDir(oldDir, newDir); err != nil {
		return err
	}
	if err := os.Remove(oldDir); err != nil {
		return fmt.Errorf("清理旧资源目录失败: %w", err)
	}
	return nil
}

// mergeStudioDir 合并目录树，并将冲突的旧文件另存为可辨认的迁移备份。
func mergeStudioDir(oldDir, newDir string) error {
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return fmt.Errorf("读取旧资源目录失败: %w", err)
	}
	for _, entry := range entries {
		oldPath := filepath.Join(oldDir, entry.Name())
		newPath := filepath.Join(newDir, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(newPath, 0o755); err != nil {
				return fmt.Errorf("创建迁移目录失败: %w", err)
			}
			if err := mergeStudioDir(oldPath, newPath); err != nil {
				return err
			}
			if err := os.Remove(oldPath); err != nil {
				return fmt.Errorf("清理已迁移目录失败: %w", err)
			}
			continue
		}
		if _, err := os.Lstat(newPath); err == nil {
			newPath += ".from-vdub_studio"
			if _, err := os.Lstat(newPath); err == nil {
				return fmt.Errorf("迁移备份目标已存在: %s", newPath)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("检查迁移备份目标失败: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("检查迁移目标失败: %w", err)
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("迁移资源文件失败: %w", err)
		}
	}
	return nil
}

// ResolveDSN 将相对路径 DSN 解析为基于 DataDir 的绝对路径
func ResolveDSN(dsn string) string {
	if filepath.IsAbs(dsn) || len(dsn) > 10 {
		return dsn
	}
	return filepath.Join(DataDir(), dsn)
}

// ApplyEnvOverrides 从环境变量覆盖敏感配置，避免将密钥提交到代码
//
// LLM:      VDUB_LLM_BASE_URL, VDUB_LLM_API_KEY, VDUB_LLM_MODEL
// Whisper:  VDUB_WHISPER_MODEL, VDUB_WHISPER_MODE
func (bc *Bootstrap) ApplyEnvOverrides() {
	if v := os.Getenv("VDUB_LLM_BASE_URL"); v != "" {
		bc.LLM.BaseURL = v
	}
	if v := os.Getenv("VDUB_LLM_API_KEY"); v != "" {
		bc.LLM.APIKey = v
	}
	if v := os.Getenv("VDUB_LLM_MODEL"); v != "" {
		bc.LLM.Model = v
	}
	if v := os.Getenv("VDUB_TRANSLATE_PROVIDER"); v != "" {
		bc.LLM.Provider = v
	}
	if v := os.Getenv("VDUB_DEEPLX_URL"); v != "" {
		bc.LLM.DeepLXURL = v
	}
	if v := os.Getenv("VDUB_WHISPER_MODEL"); v != "" {
		bc.Pipeline.WhisperModel = v
	}
	if v := os.Getenv("VDUB_WHISPER_MODE"); v != "" {
		bc.Pipeline.WhisperMode = v
	}
}

// VoiceForEngine 只使用所选引擎的音色，避免 Edge 的默认音色覆盖 OpenAI 配置。
func VoiceForEngine(config TTS, engine string) string {
	if strings.EqualFold(engine, "openai") {
		if config.OpenAIVoice != "" {
			return config.OpenAIVoice
		}
		if config.Type == "" || strings.EqualFold(config.Type, "openai") {
			return config.Voice
		}
		return ""
	}
	if config.EdgeVoice != "" {
		return config.EdgeVoice
	}
	if !strings.EqualFold(config.Type, "openai") && config.Voice != "" {
		return config.Voice
	}
	return "zh-CN-XiaoxiaoNeural"
}
