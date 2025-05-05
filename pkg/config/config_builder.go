package config

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigBuilder はディレクトリ構造からConfigを構築するための構造体です。
type ConfigBuilder struct {
	RootDir string
}

// NewConfigBuilder はConfigBuilderを生成します。
func NewConfigBuilder(rootDir string) *ConfigBuilder {
	return &ConfigBuilder{RootDir: rootDir}
}

// Build はディレクトリ構造からConfigを構築し、io.WriterにYAMLで出力します。
func (b *ConfigBuilder) Build(w io.Writer) error {
	cfg, err := b.Compile()
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(w)
	defer enc.Close()
	return enc.Encode(cfg)
}

// Compile はディレクトリ構造からConfigを構築します。
func (b *ConfigBuilder) Compile() (*Config, error) {
	// ルートのmetadata.yamlを読む
	rootMeta, err := readMetadata(filepath.Join(b.RootDir, "metadata.yaml"))
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	// actions
	if acts, ok := rootMeta["actions"]; ok {
		actsYaml, _ := yaml.Marshal(acts)
		yaml.Unmarshal(actsYaml, &cfg.Actions)
	}
	// tools
	if tools, ok := rootMeta["tools"]; ok {
		var toolDefs []map[string]interface{}
		toolsYaml, _ := yaml.Marshal(tools)
		yaml.Unmarshal(toolsYaml, &toolDefs)
		for _, t := range toolDefs {
			if path, ok := t["path"].(string); ok {
				tool, err := buildTool(filepath.Join(b.RootDir, path))
				if err != nil {
					return nil, err
				}
				cfg.Tools = append(cfg.Tools, *tool)
			}
		}
	}
	return cfg, nil
}

// readMetadata はmetadata.yamlを読み込んでmapで返します。
func readMetadata(path string) (map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var m map[string]interface{}
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// buildTool はツールディレクトリからTool構造体を構築します。
func buildTool(dir string) (*Tool, error) {
	meta, err := readMetadata(filepath.Join(dir, "metadata.yaml"))
	if err != nil {
		return nil, err
	}
	tool := &Tool{}
	// name
	tool.Name = filepath.Base(dir)
	// params
	if params, ok := meta["params"]; ok {
		paramsYaml, _ := yaml.Marshal(params)
		yaml.Unmarshal(paramsYaml, &tool.Params)
	}
	// script
	if script, ok := meta["script"].(string); ok {
		scriptPath := filepath.Join(dir, script)
		if content, err := os.ReadFile(scriptPath); err == nil {
			tool.Script = string(content)
		} else {
			tool.Script = script // ファイルがなければそのまま
		}
	}
	// beforeExec/afterExec
	if be, ok := meta["beforeExec"]; ok {
		var beList []map[string]interface{}
		beYaml, _ := yaml.Marshal(be)
		yaml.Unmarshal(beYaml, &beList)
		for _, item := range beList {
			if path, ok := item["path"].(string); ok {
				bePath := filepath.Join(dir, path)
				if content, err := os.ReadFile(bePath); err == nil {
					tool.BeforeExec = append(tool.BeforeExec, string(content))
				} else {
					tool.BeforeExec = append(tool.BeforeExec, path)
				}
			}
		}
	}
	if ae, ok := meta["afterExec"]; ok {
		var aeList []map[string]interface{}
		aeYaml, _ := yaml.Marshal(ae)
		yaml.Unmarshal(aeYaml, &aeList)
		for _, item := range aeList {
			if path, ok := item["path"].(string); ok {
				aePath := filepath.Join(dir, path)
				if content, err := os.ReadFile(aePath); err == nil {
					tool.AfterExec = append(tool.AfterExec, string(content))
				} else {
					tool.AfterExec = append(tool.AfterExec, path)
				}
			}
		}
	}
	// subtools
	if subs, ok := meta["tools"]; ok {
		var subDefs []map[string]interface{}
		subsYaml, _ := yaml.Marshal(subs)
		yaml.Unmarshal(subsYaml, &subDefs)
		for _, s := range subDefs {
			if path, ok := s["path"].(string); ok {
				sub, err := buildSubtool(filepath.Join(dir, path))
				if err != nil {
					return nil, err
				}
				tool.Subtools = append(tool.Subtools, *sub)
			}
		}
	}
	return tool, nil
}

// buildSubtool はサブツールディレクトリからSubtool構造体を構築します。
func buildSubtool(dir string) (*Subtool, error) {
	meta, err := readMetadata(filepath.Join(dir, "metadata.yaml"))
	if err != nil {
		return nil, err
	}
	sub := &Subtool{}
	sub.Name = filepath.Base(dir)
	if params, ok := meta["params"]; ok {
		paramsYaml, _ := yaml.Marshal(params)
		yaml.Unmarshal(paramsYaml, &sub.Params)
	}
	if script, ok := meta["script"].(string); ok {
		scriptPath := filepath.Join(dir, script)
		if content, err := os.ReadFile(scriptPath); err == nil {
			sub.Script = string(content)
		} else {
			sub.Script = script
		}
	}
	if be, ok := meta["beforeExec"]; ok {
		var beList []map[string]interface{}
		beYaml, _ := yaml.Marshal(be)
		yaml.Unmarshal(beYaml, &beList)
		for _, item := range beList {
			if path, ok := item["path"].(string); ok {
				bePath := filepath.Join(dir, path)
				if content, err := os.ReadFile(bePath); err == nil {
					sub.BeforeExec = append(sub.BeforeExec, string(content))
				} else {
					sub.BeforeExec = append(sub.BeforeExec, path)
				}
			}
		}
	}
	if ae, ok := meta["afterExec"]; ok {
		var aeList []map[string]interface{}
		aeYaml, _ := yaml.Marshal(ae)
		yaml.Unmarshal(aeYaml, &aeList)
		for _, item := range aeList {
			if path, ok := item["path"].(string); ok {
				aePath := filepath.Join(dir, path)
				if content, err := os.ReadFile(aePath); err == nil {
					sub.AfterExec = append(sub.AfterExec, string(content))
				} else {
					sub.AfterExec = append(sub.AfterExec, path)
				}
			}
		}
	}
	// ネストしたサブツール
	if subs, ok := meta["tools"]; ok {
		var subDefs []map[string]interface{}
		subsYaml, _ := yaml.Marshal(subs)
		yaml.Unmarshal(subsYaml, &subDefs)
		for _, s := range subDefs {
			if path, ok := s["path"].(string); ok {
				subsub, err := buildSubtool(filepath.Join(dir, path))
				if err != nil {
					return nil, err
				}
				sub.Subtools = append(sub.Subtools, *subsub)
			}
		}
	}
	return sub, nil
}
