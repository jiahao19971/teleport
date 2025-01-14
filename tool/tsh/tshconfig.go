/*
Copyright 2022 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gravitational/teleport/api/profile"

	"github.com/gravitational/trace"
	"gopkg.in/yaml.v2"
)

// .tsh config must go in a subdir as all .yaml files in .tsh get
// parsed automatically by the profile loader and results in yaml
// unmarshal errors.
const tshConfigPath = "config/config.yaml"

// default location of global tsh config file.
const globalTshConfigPathDefault = "/etc/tsh.yaml"

// TshConfig represents configuration loaded from the tsh config file.
type TshConfig struct {
	// ExtraHeaders are additional http headers to be included in
	// webclient requests.
	ExtraHeaders []ExtraProxyHeaders `yaml:"add_headers,omitempty"`
}

// ExtraProxyHeaders represents the headers to include with the
// webclient.
type ExtraProxyHeaders struct {
	// Proxy is the domain of the proxy for these set of Headers, can contain globs.
	Proxy string `yaml:"proxy"`
	// Headers are the http header key values.
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Merge two configs into one. The passed in otherConfig argument has higher priority.
func (config *TshConfig) Merge(otherConfig *TshConfig) TshConfig {
	baseConfig := config
	if baseConfig == nil {
		baseConfig = &TshConfig{}
	}

	if otherConfig == nil {
		otherConfig = &TshConfig{}
	}

	newConfig := TshConfig{}

	// extra headers
	newConfig.ExtraHeaders = append(baseConfig.ExtraHeaders, otherConfig.ExtraHeaders...)

	return newConfig
}

// loadConfig load a single config file from given path. If the path does not exist, an empty config is returned instead.
func loadConfig(fullConfigPath string) (*TshConfig, error) {
	bs, err := os.ReadFile(fullConfigPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &TshConfig{}, nil
		}
		return nil, trace.ConvertSystemError(err)
	}

	cfg := TshConfig{}
	if err := yaml.Unmarshal(bs, &cfg); err != nil {
		return nil, trace.ConvertSystemError(err)
	}
	return &cfg, nil
}

// loadAllConfigs loads all tsh configs and merges them in appropriate order.
func loadAllConfigs(cf CLIConf) (*TshConfig, error) {
	// default to globalTshConfigPathDefault
	globalConfigPath := cf.GlobalTshConfigPath
	if globalConfigPath == "" {
		globalConfigPath = globalTshConfigPathDefault
	}

	globalConf, err := loadConfig(globalConfigPath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to load global tsh config from %q", cf.GlobalTshConfigPath)
	}

	fullConfigPath := filepath.Join(profile.FullProfilePath(cf.HomePath), tshConfigPath)
	userConf, err := loadConfig(fullConfigPath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to load tsh config from %q", fullConfigPath)
	}

	confOptions := globalConf.Merge(userConf)
	return &confOptions, nil
}
