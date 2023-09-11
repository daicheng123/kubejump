package config

import (
	"fmt"
	"github.com/daicheng123/kubejump/internal/entity"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

const (
	hostEnvKey = "SERVER_HOSTNAME"
)

var GlobalConfig *Config

func GetConf() Config {
	if GlobalConfig == nil {
		return getDefaultConfig()
	}
	return *GlobalConfig
}

type Config struct {
	Name        string `mapstructure:"name"`
	ServerDebug bool   `mapstructure:"server_debug"`

	BindHost   string `mapstructure:"bind_host"`
	SSHPort    string `mapstructure:"ssh_port"`
	HTTPPort   string `mapstructure:"httpd_port"`
	SSHTimeout int    `mapstructure:"ssh_timeout"`

	LogLevel string `mapstructure:"log_level"`

	ClientAliveInterval int `mapstructure:"CLIENT_ALIVE_INTERVAL"`

	DatabaseName     string `mapstructure:"database_name"`
	DatabasePort     int    `mapstructure:"database_port"`
	DatabaseAddress  string `mapstructure:"database_address"`
	DatabasePassword string `mapstructure:"database_password"`
	DatabaseUser     string `mapstructure:"database_user"`

	AssetLoadPolicy string `mapstructure:"asset_load_policy"` // all

	EnableLocalPortForward bool `mapstructure:"enable_local_port_forward"`

	RootPath       string
	LogDirPath     string
	LocalCachePath string

	TerminalConf *entity.TerminalConfig

	//DataFolderPath    string
	//KeyFolderPath     string
	//AccessKeyFilePath string
	//ReplayFolderPath  string

	//CoreHost       string `mapstructure:"CORE_HOST"`
	//BootstrapToken string `mapstructure:"BOOTSTRAP_TOKEN"`
	//Comment             string `mapstructure:"COMMENT"`
	//LanguageCode        string `mapstructure:"LANGUAGE_CODE"`
	//UploadFailedReplay  bool   `mapstructure:"UPLOAD_FAILED_REPLAY_ON_START"`
	//AssetLoadPolicy     string `mapstructure:"ASSET_LOAD_POLICY"` // all
	//ZipMaxSize          string `mapstructure:"ZIP_MAX_SIZE"`
	//ZipTmpPath          string `mapstructure:"ZIP_TMP_PATH"`
	//ClientAliveInterval int    `mapstructure:"CLIENT_ALIVE_INTERVAL"`
	//RetryAliveCountMax  int    `mapstructure:"RETRY_ALIVE_COUNT_MAX"`
	//ShowHiddenFile      bool   `mapstructure:"SFTP_SHOW_HIDDEN_FILE"`
	//ReuseConnection     bool   `mapstructure:"REUSE_CONNECTION"`
	//
	//ShareRoomType string   `mapstructure:"SHARE_ROOM_TYPE"`
	//RedisHost     string   `mapstructure:"REDIS_HOST"`
	//RedisPort     string   `mapstructure:"REDIS_PORT"`
	//RedisPassword string   `mapstructure:"REDIS_PASSWORD"`
	//RedisDBIndex  int      `mapstructure:"REDIS_DB_ROOM"`
	//RedisClusters []string `mapstructure:"REDIS_CLUSTERS"`
	//EnableVscodeSupport    bool `mapstructure:"ENABLE_VSCODE_SUPPORT"`
	//
}

func Setup(configPath string) {
	var conf = getDefaultConfig()
	loadConfigFromEnv(&conf)
	loadConfigFromFile(configPath, &conf)

	GlobalConfig = &conf
	klog.Infof("%+v\n", GlobalConfig)
}

func getDefaultConfig() Config {
	defaultName := getDefaultName()
	rootPath := getPwdDirPath()

	dataFolderPath := filepath.Join(rootPath, "data")
	localCachePath := filepath.Join(dataFolderPath, "cache.txt")

	folders := []string{dataFolderPath}
	for i := range folders {
		if err := EnsureDirExist(folders[i]); err != nil {
			klog.Fatalf("Create folder failed: %s", err.Error())
		}
	}

	return Config{
		Name:                   defaultName,
		ServerDebug:            true,
		BindHost:               "0.0.0.0",
		SSHPort:                "2222",
		HTTPPort:               "8081",
		RootPath:               rootPath,
		LogLevel:               "info",
		SSHTimeout:             15,
		EnableLocalPortForward: false,
		DatabaseName:           "devops",
		DatabaseAddress:        "127.0.0.1",
		DatabaseUser:           "root",
		DatabasePort:           3306,
		DatabasePassword:       "root",
		LocalCachePath:         localCachePath,
		AssetLoadPolicy:        "all",

		ClientAliveInterval: 120,
		// terminal 终端配置
		TerminalConf: &entity.TerminalConfig{
			AssetListPageSize: "20",
			AssetListSortBy:   "ClusterName",
			MaxIdleTime:       60,
		},
	}
}

func getDefaultName() string {
	hostname, _ := os.Hostname()
	if serverHostname, ok := os.LookupEnv(hostEnvKey); ok {
		hostname = fmt.Sprintf("%s-%s", serverHostname, hostname)
	}

	return hostname
}

func getPwdDirPath() string {
	if rootPath, err := os.Getwd(); err == nil {
		return rootPath
	}
	return ""
}

func EnsureDirExist(path string) error {
	if !haveDir(path) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

//func EnsureFileExist(path string) error {
//	if !haveDir(path) {
//		if err := os.path, os.ModePerm); err != nil {
//			return err
//		}
//	}
//	return nil
//}

func haveDir(file string) bool {
	fi, err := os.Stat(file)
	return err == nil && fi.IsDir()
}

func have(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func loadConfigFromEnv(conf *Config) {
	viper.AutomaticEnv()
	envViper := viper.New()
	for _, item := range os.Environ() {
		envItem := strings.SplitN(item, "=", 2)

		if len(envItem) == 2 {
			envViper.Set(envItem[0], viper.Get(envItem[0]))
		}
	}
	if err := envViper.Unmarshal(conf); err == nil {
		klog.Infoln("Load config from env")
	}
}

func loadConfigFromFile(path string, conf *Config) {
	var err error
	fileViper := viper.New()

	fileViper.SetConfigFile(path)

	if err = fileViper.ReadInConfig(); err == nil {
		if err = fileViper.Unmarshal(conf); err == nil {
			klog.Infof("Load config from %s success\n", path)
			return
		}
	}
	if err != nil {
		klog.Fatalf("Load config from %s failed: %s\n", path, err)
	}
}
