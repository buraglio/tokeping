package config

import (
    "time"

    "github.com/spf13/viper"
)

type ProbeConfig struct {
    Name     string        `mapstructure:"name"`
    Type     string        `mapstructure:"type"`
    Target   string        `mapstructure:"target"`
    Interval time.Duration `mapstructure:"interval"`
}

type OutputConfig struct {
    Name   string `mapstructure:"name"`
    Type   string `mapstructure:"type"`
    Listen string `mapstructure:"listen,omitempty"`
    URL    string `mapstructure:"url,omitempty"`
    Token  string `mapstructure:"token,omitempty"`
    Org    string `mapstructure:"org,omitempty"`
    Bucket string `mapstructure:"bucket,omitempty"`
    Path   string `mapstructure:"path,omitempty"`
}

type Config struct {
    Probes   []ProbeConfig  `mapstructure:"probes"`
    Outputs  []OutputConfig `mapstructure:"outputs"`
    PIDFile  string         `mapstructure:"pid_file,omitempty"`
}

func Load(path string) (*Config, error) {
    viper.SetConfigFile(path)
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
