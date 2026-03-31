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
	Polymarket     Polymarket     `yaml:"polymarket"`     // polymarket（只读同步）
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

// Polymarket 同步配置（只读）
// - 只同步指定 tags / market slugs
// - 不同步价格盘口（不接 CLOB），只同步市场目录与最终结算结果（resolved outcome）
type Polymarket struct {
	Enabled     bool     `yaml:"enabled"`
	BaseURL     string   `yaml:"baseURL"`     // Gamma API base url，留空用默认：https://gamma-api.polymarket.com
	CronSpec    string   `yaml:"cronSpec"`    // 定时同步 cron（5字段），默认：*/30 * * * *
	Tags        []string `yaml:"tags"`        // 只同步这些 tag slug（小写）
	MarketSlugs []string `yaml:"marketSlugs"` // 额外同步指定 market slug 白名单
	PageSize    int      `yaml:"pageSize"`    // 分页 size，默认 50
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
	}

	slog.Info("Load config", slog.String("ENV", GetEnv()))
	return cfg, exists, nil
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

func defaultDbConfig() sqls.DbConfig {
	return sqls.DbConfig{
		Type:                   DbTypePostgres,
		MaxIdleConns:           50,
		MaxOpenConns:           200,
		ConnMaxIdleTimeSeconds: 300,
		ConnMaxLifetimeSeconds: 3600,
	}
}
