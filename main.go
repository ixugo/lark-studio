package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ixugo/goddd/domain/version/versionapi"
	"github.com/ixugo/goddd/pkg/system"
	"github.com/ixugo/vdub/internal/app"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
)

var (
	buildVersion = "0.0.1" // 构建版本号
	gitBranch    = "dev"   // git 分支
	gitHash      = "debug" // git 提交点哈希值
	release      string    // 发布模式 true/false
	buildTime    string    // 构建时间戳
)

var (
	configDir = flag.String("conf", "./configs", "config directory, eg: -conf /configs/")
	runFile   = flag.String("run", "", "directly run pipeline on a video file, skip HTTP server")
	runMode   = flag.Int("mode", 2, "processing mode: 1=subtitle, 2=translate, 3=dub")
	runLang   = flag.String("lang", "", "target language (default from config)")
	runSS     = flag.Float64("ss", 0, "clip start time in seconds")
	runTo     = flag.Float64("to", 0, "clip end time in seconds")
	runOutput = flag.String("output", "", "output directory (default: video dir + _vdub)")
)

func getBuildRelease() bool {
	v, _ := strconv.ParseBool(release)
	return v
}

func main() {
	flag.Parse()

	// 初始化配置
	var bc conf.Bootstrap
	fileDir, _ := system.Abs(*configDir)
	_ = os.MkdirAll(fileDir, 0o755)
	filePath := filepath.Join(fileDir, "config.toml")
	configIsNotExistWrite(filePath)
	if err := conf.SetupConfig(&bc, filePath); err != nil {
		panic(err)
	}
	bc.ApplyEnvOverrides()
	bc.Runtime.Debug = !getBuildRelease()
	bc.Runtime.BuildVersion = buildVersion
	bc.Runtime.ConfigDir = fileDir
	bc.Runtime.ConfigPath = filePath

	// CLI 模式：直接处理单个视频
	if *runFile != "" {
		runCLI(&bc)
		return
	}

	{
		expvar.NewString("version").Set(buildVersion)
		expvar.NewString("git_branch").Set(gitBranch)
		expvar.NewString("git_hash").Set(gitHash)
		expvar.NewString("build_time").Set(buildTime)
		expvar.Publish("timestamp", expvar.Func(func() any {
			return time.Now().Format(time.DateTime)
		}))
	}

	versionapi.DBVersion = buildVersion
	versionapi.DBRemark = gitBranch + "_" + gitHash

	app.Run(&bc)
}

// runCLI 终端直接运行流水线，不启动 HTTP 服务
func runCLI(bc *conf.Bootstrap) {
	inputPath := *runFile
	if _, err := os.Stat(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "视频文件不存在: %s\n", inputPath)
		os.Exit(1)
	}

	// 确定输出目录
	outputDir := *runOutput
	if outputDir == "" {
		baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outputDir = filepath.Join(filepath.Dir(inputPath), baseName+"_vdub")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %s\n", err)
		os.Exit(1)
	}

	// ffmpeg 裁剪（可选）
	actualInput := inputPath
	if *runSS > 0 || *runTo > 0 {
		clipped, err := clipVideo(bc.Pipeline.FFmpegBin, inputPath, outputDir, *runSS, *runTo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "裁剪视频失败: %s\n", err)
			os.Exit(1)
		}
		actualInput = clipped
		slog.Info("视频已裁剪", "output", clipped)
	}

	// 目标语言
	lang := *runLang
	if lang == "" {
		lang = bc.Pipeline.DefaultTargetLang
	}

	// 构建流水线
	pipeCore := app.NewPipelineCore(bc)
	job := pipeline.Job{
		TaskID:     fmt.Sprintf("cli_%d", time.Now().Unix()),
		InputPath:  actualInput,
		OutputDir:  outputDir,
		Mode:       *runMode,
		TargetLang: lang,
	}

	slog.Info("开始处理",
		"input", actualInput,
		"output", outputDir,
		"mode", job.Mode,
		"lang", lang,
	)

	ctx := context.Background()
	if err := pipeCore.Run(ctx, job); err != nil {
		fmt.Fprintf(os.Stderr, "流水线执行失败: %s\n", err)
		os.Exit(1)
	}

	slog.Info("处理完成", "output", outputDir)
}

// clipVideo 用 ffmpeg 裁剪视频片段
func clipVideo(ffmpegBin, inputPath, outputDir string, ss, to float64) (string, error) {
	if ffmpegBin == "" {
		ffmpegBin = "ffmpeg"
	}

	clippedPath := filepath.Join(outputDir, "clipped"+filepath.Ext(inputPath))
	args := []string{"-y", "-i", inputPath}
	if ss > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", ss))
	}
	if to > 0 {
		args = append(args, "-to", fmt.Sprintf("%.3f", to))
	}
	args = append(args, "-c", "copy", clippedPath)

	cmd := exec.Command(ffmpegBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s\noutput: %s", err, string(output))
	}
	return clippedPath, nil
}

// configIsNotExistWrite 配置文件不存在时，回写配置
func configIsNotExistWrite(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := conf.WriteConfig(conf.DefaultConfig(), path); err != nil {
			system.ErrPrintf("WriteConfig", "err", err)
		}
	}
}
