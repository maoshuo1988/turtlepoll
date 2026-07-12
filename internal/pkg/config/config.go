package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	BBSGO_ENV  = "BBSGO_ENV"
	ENV_PREFIX = "BBSGO"

	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Language string

const (
	LanguageZhCN Language = "zh-CN"
	LanguageEnUS Language = "en-US"

	DefaultLanguage = LanguageEnUS
)

func (l Language) IsValid() bool {
	switch l {
	case LanguageZhCN, LanguageEnUS:
		return true
	}
	return false
}

var (
	Instance   *Config
	v          *viper.Viper
	configFile string
	writeMx    sync.Mutex
)

func init() {
	var (
		configFileName = "bbs-go.yaml"
	)
	v = viper.New()
	v.SetConfigFile(configFileName)
	v.AddConfigPath(".")
	if workDir, err := os.Executable(); err == nil {
		v.AddConfigPath(filepath.Dir(workDir))
	}
	v.AutomaticEnv()
	v.SetEnvPrefix(ENV_PREFIX)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	configFile = getConfigFilePath(configFileName)
}

type Config struct {
	Language       Language       `yaml:"language"`       // 语言
	Port           int            `yaml:"port"`           // 端口
	IPLocator      IPLocator      `yaml:"ipLocator"`      // IP定位配置
	AllowedOrigins []string       `yaml:"allowedOrigins"` // 跨域白名单
	Installed      bool           `yaml:"installed"`      // 是否已安装
	IDCodec        IDCodecConfig  `yaml:"idCodec"`        // ID 编解码配置
	Logger         LoggerConfig   `yaml:"logger"`         // 日志配置
	DB             sqls.DbConfig  `yaml:"db"`             // 数据库配置
	Smtp           SmtpConfig     `yaml:"smtp"`           // smtp
	Search         SearchConfig   `yaml:"search"`         // 搜索配置
	BaiduSEO       BaiduSEOConfig `yaml:"baiduSEO"`       // 百度SEO配置
	SmSEO          SmSEOConfig    `yaml:"smSEO"`          // 神马搜索SEO配置
	FootballData   FootballData   `yaml:"footballData"`   // football-data.org
	News           NewsConfig     `yaml:"news"`           // 虎扑资讯采集
	Polymarket     Polymarket     `yaml:"polymarket"`     // polymarket（只读同步）
	DeepSeek       DeepSeek       `yaml:"deepseek"`       // DeepSeek API
	AIChat         AIChat         `yaml:"aiChat"`         // AI 聊天
	MessageNotify  MessageNotify  `yaml:"messageNotify"`  // 主站消息通知
	LoginCaptcha   LoginCaptcha   `yaml:"loginCaptcha"`   // 登录/注册相关验证码开关（仅配置文件层面）
}

// LoginCaptcha 登录/认证相关验证码配置
// 说明：这是“配置文件”层面的开关（bbs-go.yaml），不走 sys_config 表。
// - RotateEnabled=false 时：不再使用旋转验证码（captchaProtocol=2）的校验。
// - 若你希望“完全免验证码”，可以保持 RotateEnabled=false，且前端不再传 captchaId/captchaCode。
//
// 注意：当前登录接口对 captchaProtocol!=2 的情况会走字符验证码校验；因此这里额外提供
// DisableAllWhenRotateOff 来支持“关闭 rotate 后直接登录”的诉求（用户名密码通过即可）。
type LoginCaptcha struct {
	RotateEnabled           bool `yaml:"rotateEnabled"`           // 是否启用旋转验证码（captchaProtocol=2）
	DisableAllWhenRotateOff bool `yaml:"disableAllWhenRotateOff"` // 关闭 rotate 时，是否同时跳过所有登录/注册相关验证码校验
}

type FootballData struct {
	APIKey          string `yaml:"apiKey"`
	BaseURL         string `yaml:"baseURL"`
	CompetitionCode string `yaml:"competitionCode"` // e.g. WC
	Season          int    `yaml:"season"`          // e.g. 2026
	CronSpec        string `yaml:"cronSpec"`        // e.g. "0 */30 * * * *" (every 30 min)
}

type NewsConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	Source                  string `yaml:"source"`
	BaseURL                 string `yaml:"baseURL"`
	BbsBaseURL              string `yaml:"bbsBaseURL"`
	CronSpec                string `yaml:"cronSpec"`
	BatchSize               int    `yaml:"batchSize"`
	RequestTimeoutSeconds   int    `yaml:"requestTimeoutSeconds"`
	MaxQPSPerWorker         int    `yaml:"maxQPSPerWorker"`
	MaxConcurrencyPerDomain int    `yaml:"maxConcurrencyPerDomain"`
	MaxRetry                int    `yaml:"maxRetry"`
	RetryBaseSeconds        int    `yaml:"retryBaseSeconds"`
	RetryMaxSeconds         int    `yaml:"retryMaxSeconds"`
	CircuitBreakMinutes     int    `yaml:"circuitBreakMinutes"`
	AllowDetailRefresh      bool   `yaml:"allowDetailRefresh"`
	AllowListHot            bool   `yaml:"allowListHot"`
	AllowSearch             bool   `yaml:"allowSearch"`
}

// Polymarket 同步配置（只读）
// - 标签来源：数据库表 t_polymarket_discovery_tag（由 migration 预置）
// - 不同步价格盘口（不接 CLOB），只同步市场目录与最终结算结果（resolved outcome）
type Polymarket struct {
	Enabled           bool   `yaml:"enabled"`
	BaseURL           string `yaml:"baseURL"`       // Gamma API base url，留空用默认：https://gamma-api.polymarket.com
	CronSpec          string `yaml:"cronSpec"`      // 兼容旧配置：Discovery 定时同步 cron
	DiscoveryCron     string `yaml:"discoveryCron"` // Discovery cron，默认：*/30 * * * *
	TrackingCron      string `yaml:"trackingCron"`  // Tracking cron，默认：*/5 * * * *
	PageSize          int    `yaml:"pageSize"`      // 兼容旧配置：Discovery 分页 size，默认 50
	TrackingBatchSize int    `yaml:"trackingBatchSize"`
	MaxRetry          int    `yaml:"maxRetry"`
	RetryBaseSeconds  int    `yaml:"retryBaseSeconds"`
	AutoSettleEnabled *bool  `yaml:"autoSettleEnabled"`
	DryRun            bool   `yaml:"dryRun"`
}

type DeepSeek struct {
	Enabled        bool   `yaml:"enabled"`
	BaseURL        string `yaml:"baseURL"`
	APIKey         string `yaml:"apiKey"`
	DefaultModel   string `yaml:"defaultModel"`
	ReasoningModel string `yaml:"reasoningModel"`
	TimeoutSeconds int    `yaml:"timeoutSeconds"`
	MaxRetries     int    `yaml:"maxRetries"`
}

type AIChat struct {
	Enabled                 bool `yaml:"enabled"`
	DefaultStaminaCost      int  `yaml:"defaultStaminaCost"`
	DefaultMaxStamina       int  `yaml:"defaultMaxStamina"`
	StaminaRecoverMinutes   int  `yaml:"staminaRecoverMinutes"`
	AppleCoinCost           int  `yaml:"appleCoinCost"`
	MaxInputChars           int  `yaml:"maxInputChars"`
	MaxHistoryMessages      int  `yaml:"maxHistoryMessages"`
	DailyUserMessageLimit   int  `yaml:"dailyUserMessageLimit"`
	IdlePushCooldownMinutes int  `yaml:"idlePushCooldownMinutes"`
	IdlePushDailyLimit      int  `yaml:"idlePushDailyLimit"`
	IdleTriggerMinutes      int  `yaml:"idleTriggerMinutes"`
}

type MessageNotify struct {
	Enabled                   *bool `yaml:"enabled"`
	DefaultPageSize           int   `yaml:"defaultPageSize"`
	MaxPageSize               int   `yaml:"maxPageSize"`
	RenderStrict              *bool `yaml:"renderStrict"`
	ReadUpdateThrottleSeconds int   `yaml:"readUpdateThrottleSeconds"`
}

type IPLocator struct {
	IPv4DataPath string `yaml:"ipv4DataPath"` // IPv4 数据文件路径
	IPv6DataPath string `yaml:"ipv6DataPath"` // IPv6 数据文件路径
}

type IDCodecConfig struct {
	Key uint64 `yaml:"key"` // ID 编解码秘钥
}

type LoggerConfig struct {
	Filename   string `yaml:"filename"`   // 日志文件的位置
	MaxSize    int    `yaml:"maxSize"`    // 文件最大尺寸（以MB为单位）
	MaxAge     int    `yaml:"maxAge"`     // 保留旧文件的最大天数
	MaxBackups int    `yaml:"maxBackups"` // 保留的最大旧文件数量
}

type SmtpConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSL      bool   `yaml:"ssl"`
}

type SearchConfig struct {
	IndexPath string `yaml:"indexPath"`
}

// 百度SEO配置
// 文档：https://ziyuan.baidu.com/college/courseinfo?id=267&page=2#h2_article_title14
type BaiduSEOConfig struct {
	Site  string `yaml:"site"`
	Token string `yaml:"token"`
}

// 神马搜索SEO配置
// 文档：https://zhanzhang.sm.cn/open/mip
type SmSEOConfig struct {
	Site     string `yaml:"site"`
	UserName string `yaml:"userName"`
	Token    string `yaml:"token"`
}

func ReadConfig() (cfg *Config, exists bool, err error) {
	exists = true
	if e := v.ReadInConfig(); e != nil {
		exists = false
		slog.Warn("Config file not found, use default", slog.Any("error", e))
	}

	if exists {
		if e := v.Unmarshal(&cfg); e != nil {
			err = fmt.Errorf("fatal error unmarshal config: %w", err)
			return
		}
		// 如果配置文件存在但没有语言设置，使用默认语言
		if strs.IsBlank(string(cfg.Language)) {
			cfg.Language = DefaultLanguage
		}
		SetDbDefaults(&cfg.DB)
		SetAIDefaults(cfg)
		SetMessageNotifyDefaults(cfg)
		SetNewsDefaults(cfg)
		SetPolymarketDefaults(cfg)
	} else {
		// default config
		cfg = &Config{
			Language:  DefaultLanguage,
			Port:      8082,
			Installed: false,
			Logger: LoggerConfig{
				Filename:   getLogFilename(),
				MaxSize:    10,
				MaxAge:     10,
				MaxBackups: 10,
			},
			DB: defaultDbConfig(),
		}
		SetAIDefaults(cfg)
		SetMessageNotifyDefaults(cfg)
		SetNewsDefaults(cfg)
		SetPolymarketDefaults(cfg)
	}

	slog.Info("Load config", slog.String("ENV", GetEnv()))
	return cfg, exists, nil
}

func SetPolymarketDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Polymarket.AutoSettleEnabled == nil {
		enabled := true
		cfg.Polymarket.AutoSettleEnabled = &enabled
	}
}

func SetNewsDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	hasNewsSection := cfg.News.Source != "" || cfg.News.BaseURL != "" || cfg.News.BbsBaseURL != "" || cfg.News.CronSpec != "" || cfg.News.BatchSize > 0
	if cfg.News.Source == "" {
		cfg.News.Source = "hupu"
	}
	if cfg.News.BaseURL == "" {
		cfg.News.BaseURL = "https://www.hupu.com"
	}
	if cfg.News.BbsBaseURL == "" {
		cfg.News.BbsBaseURL = "https://bbs.hupu.com"
	}
	if cfg.News.CronSpec == "" {
		cfg.News.CronSpec = "*/10 * * * *"
	}
	if cfg.News.BatchSize <= 0 {
		cfg.News.BatchSize = 50
	}
	if cfg.News.RequestTimeoutSeconds <= 0 {
		cfg.News.RequestTimeoutSeconds = 5
	}
	if cfg.News.MaxQPSPerWorker <= 0 {
		cfg.News.MaxQPSPerWorker = 1
	}
	if cfg.News.MaxConcurrencyPerDomain <= 0 {
		cfg.News.MaxConcurrencyPerDomain = 2
	}
	if cfg.News.MaxRetry <= 0 {
		cfg.News.MaxRetry = 3
	}
	if cfg.News.RetryBaseSeconds <= 0 {
		cfg.News.RetryBaseSeconds = 60
	}
	if cfg.News.RetryMaxSeconds <= 0 {
		cfg.News.RetryMaxSeconds = 120
	}
	if cfg.News.CircuitBreakMinutes <= 0 {
		cfg.News.CircuitBreakMinutes = 15
	}
	if !hasNewsSection {
		// 配置缺失时默认启用。
		cfg.News.Enabled = true
		cfg.News.AllowDetailRefresh = true
		cfg.News.AllowListHot = true
		cfg.News.AllowSearch = true
	}
}

func WriteConfig(cfg *Config) error {
	if !writeMx.TryLock() {
		return errors.New("config is being written, please try again later")
	}
	defer writeMx.Unlock()

	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	slog.Info("Write config", slog.String("configFile", configFile))

	err = os.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func IsProd() bool {
	e := strings.ToLower(GetEnv())
	return e == "prod" || e == "production"
}

func GetEnv() string {
	env := os.Getenv("BBSGO_ENV")
	if strs.IsBlank(env) {
		env = EnvDev
	}
	return env
}

func getConfigFilePath(configName string) string {
	// Always prefer writing next to the working directory, even when the file does not yet exist.
	cwdPath := filepath.Join(".", configName)
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath
	}
	// If CWD is accessible but file is missing, still choose CWD so installs do not drift to temp dirs.
	if _, err := os.Stat("."); err == nil {
		return cwdPath
	}

	// Fallbacks: first try beside the executable if reachable, otherwise return the bare name.
	if workDir, err := os.Executable(); err == nil {
		exePath := filepath.Join(filepath.Dir(workDir), configName)
		if _, err := os.Stat(exePath); err == nil {
			return exePath
		}
		return exePath
	}
	return configName
}

func GetConfigDir() string {
	return filepath.Dir(configFile)
}

func getLogFilename() string {
	// workDir, err := os.Getwd()
	// if err != nil {
	// 	slog.Error("Failed to get working directory", slog.Any("error", err))
	// 	return ""
	// }
	return filepath.Join("./", "logs", "bbs-go.log")
}

const (
	DbTypePostgres = "postgres"
)

func SetDbDefaults(c *sqls.DbConfig) {
	if c.Type == "" || c.Type != DbTypePostgres {
		c.Type = DbTypePostgres
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 50
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 200
	}
	if c.ConnMaxIdleTimeSeconds == 0 {
		c.ConnMaxIdleTimeSeconds = 300
	}
	if c.ConnMaxLifetimeSeconds == 0 {
		c.ConnMaxLifetimeSeconds = 3600
	}
}

func SetAIDefaults(c *Config) {
	if c == nil {
		return
	}
	if c.DeepSeek.BaseURL == "" {
		c.DeepSeek.BaseURL = "https://api.deepseek.com"
	}
	if c.DeepSeek.DefaultModel == "" {
		c.DeepSeek.DefaultModel = "deepseek-v4-flash"
	}
	if c.DeepSeek.ReasoningModel == "" {
		c.DeepSeek.ReasoningModel = "deepseek-v4-pro"
	}
	if c.DeepSeek.TimeoutSeconds == 0 {
		c.DeepSeek.TimeoutSeconds = 120
	}
	if c.DeepSeek.MaxRetries == 0 {
		c.DeepSeek.MaxRetries = 2
	}
	if env := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); env != "" {
		c.DeepSeek.APIKey = env
	}
	if env := strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL")); env != "" {
		c.DeepSeek.BaseURL = env
	}
	if env := strings.TrimSpace(os.Getenv("DEEPSEEK_DEFAULT_MODEL")); env != "" {
		c.DeepSeek.DefaultModel = env
	}
	if c.AIChat.DefaultStaminaCost == 0 {
		c.AIChat.DefaultStaminaCost = 1
	}
	if c.AIChat.DefaultMaxStamina == 0 {
		c.AIChat.DefaultMaxStamina = 5
	}
	if c.AIChat.StaminaRecoverMinutes == 0 {
		c.AIChat.StaminaRecoverMinutes = 60
	}
	if c.AIChat.AppleCoinCost == 0 {
		c.AIChat.AppleCoinCost = 5
	}
	if c.AIChat.MaxInputChars == 0 {
		c.AIChat.MaxInputChars = 500
	}
	if c.AIChat.MaxHistoryMessages == 0 {
		c.AIChat.MaxHistoryMessages = 8
	}
	if c.AIChat.DailyUserMessageLimit == 0 {
		c.AIChat.DailyUserMessageLimit = 50
	}
	if c.AIChat.IdlePushCooldownMinutes == 0 {
		c.AIChat.IdlePushCooldownMinutes = 120
	}
	if c.AIChat.IdlePushDailyLimit == 0 {
		c.AIChat.IdlePushDailyLimit = 2
	}
	if c.AIChat.IdleTriggerMinutes == 0 {
		c.AIChat.IdleTriggerMinutes = 10
	}
}

func SetMessageNotifyDefaults(c *Config) {
	if c == nil {
		return
	}
	if c.MessageNotify.DefaultPageSize == 0 {
		c.MessageNotify.DefaultPageSize = 20
	}
	if c.MessageNotify.MaxPageSize == 0 {
		c.MessageNotify.MaxPageSize = 100
	}
	if c.MessageNotify.MaxPageSize < c.MessageNotify.DefaultPageSize {
		c.MessageNotify.MaxPageSize = c.MessageNotify.DefaultPageSize
	}
	if c.MessageNotify.Enabled == nil {
		enabled := true
		c.MessageNotify.Enabled = &enabled
	}
	if c.MessageNotify.RenderStrict == nil {
		renderStrict := true
		c.MessageNotify.RenderStrict = &renderStrict
	}
}

func defaultDbConfig() sqls.DbConfig {
	return sqls.DbConfig{
		Type:                   DbTypePostgres,
		MaxIdleConns:           50,
		MaxOpenConns:           200,
		ConnMaxIdleTimeSeconds: 300,
		ConnMaxLifetimeSeconds: 3600,
	}
}
