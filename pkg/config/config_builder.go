package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	// |4 などを |- に置換
	re := regexp.MustCompile(`\|[0-9]+`)
	fixed := re.ReplaceAll(out, []byte("|-"))
	_, err = w.Write(fixed)
	return err
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
		if err := yaml.Unmarshal(actsYaml, &cfg.Actions); err != nil {
			return nil, err
		}
	}
	// tools
	if tools, ok := rootMeta["tools"]; ok {
		var toolDefs []map[string]interface{}
		toolsYaml, _ := yaml.Marshal(tools)
		if err := yaml.Unmarshal(toolsYaml, &toolDefs); err != nil {
			return nil, err
		}
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
	// description
	if description, ok := meta["description"].(string); ok {
		tool.Description = description
	}
	// params
	if params, ok := meta["params"]; ok {
		paramsYaml, _ := yaml.Marshal(params)
		if err := yaml.Unmarshal(paramsYaml, &tool.Params); err != nil {
			return nil, err
		}
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
		if err := yaml.Unmarshal(beYaml, &beList); err != nil {
			return nil, err
		}
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
		if err := yaml.Unmarshal(aeYaml, &aeList); err != nil {
			return nil, err
		}
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
		if err := yaml.Unmarshal(subsYaml, &subDefs); err != nil {
			return nil, err
		}
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
	if description, ok := meta["description"].(string); ok {
		sub.Description = description
	}
	if params, ok := meta["params"]; ok {
		paramsYaml, _ := yaml.Marshal(params)
		if err := yaml.Unmarshal(paramsYaml, &sub.Params); err != nil {
			return nil, err
		}
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
		if err := yaml.Unmarshal(beYaml, &beList); err != nil {
			return nil, err
		}
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
		if err := yaml.Unmarshal(aeYaml, &aeList); err != nil {
			return nil, err
		}
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
	// param_refs
	if paramRefs, ok := meta["param_refs"]; ok {
		paramRefsYaml, _ := yaml.Marshal(paramRefs)
		if err := yaml.Unmarshal(paramRefsYaml, &sub.ParamRefs); err != nil {
			return nil, err
		}
	}
	// danger_level
	if dangerLevel, ok := meta["danger_level"].(string); ok {
		sub.DangerLevel = dangerLevel
	}
	// ネストしたサブツール
	if subs, ok := meta["tools"]; ok {
		var subDefs []map[string]interface{}
		subsYaml, _ := yaml.Marshal(subs)
		if err := yaml.Unmarshal(subsYaml, &subDefs); err != nil {
			return nil, err
		}
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

// ExportToDir はConfig構造体をディレクトリ構成に展開します。
func (b *ConfigBuilder) ExportToDir(cfg *Config, outDir string) error {
	// ルートmetadata.yaml作成
	rootMeta := map[string]interface{}{}
	if len(cfg.Actions) > 0 {
		rootMeta["actions"] = cfg.Actions
	}
	if len(cfg.Tools) > 0 {
		var toolsList []map[string]interface{}
		for _, tool := range cfg.Tools {
			toolsList = append(toolsList, map[string]interface{}{"path": "tools/" + safeDirName(tool.Name)})
		}
		rootMeta["tools"] = toolsList
	}
	if err := writeMetadata(filepath.Join(outDir, "metadata.yaml"), rootMeta); err != nil {
		return err
	}
	// tools ディレクトリ作成
	toolsDir := filepath.Join(outDir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return err
	}
	for _, tool := range cfg.Tools {
		if err := exportTool(&tool, filepath.Join(toolsDir, safeDirName(tool.Name))); err != nil {
			return err
		}
	}
	return nil
}

func writeMetadata(path string, meta map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	defer enc.Close()
	return enc.Encode(meta)
}

func exportTool(tool *Tool, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	meta := map[string]interface{}{}
	if len(tool.Params) > 0 {
		meta["params"] = tool.Params
	}
	if tool.Script != "" {
		meta["script"] = "main.sh"
		if err := os.WriteFile(filepath.Join(dir, "main.sh"), []byte(tool.Script), 0755); err != nil {
			return err
		}
	}
	if len(tool.BeforeExec) > 0 {
		var beList []map[string]interface{}
		for i, content := range tool.BeforeExec {
			name := "beforeExec_%02d.sh"
			fname := filepath.Join(dir, "beforeExec", fmt.Sprintf(name, i))
			if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fname, []byte(content), 0755); err != nil {
				return err
			}
			beList = append(beList, map[string]interface{}{"path": filepath.Join("beforeExec", fmt.Sprintf(name, i))})
		}
		meta["beforeExec"] = beList
	}
	if len(tool.AfterExec) > 0 {
		var aeList []map[string]interface{}
		for i, content := range tool.AfterExec {
			name := "afterExec_%02d.sh"
			fname := filepath.Join(dir, "afterExec", fmt.Sprintf(name, i))
			if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fname, []byte(content), 0755); err != nil {
				return err
			}
			aeList = append(aeList, map[string]interface{}{"path": filepath.Join("afterExec", fmt.Sprintf(name, i))})
		}
		meta["afterExec"] = aeList
	}
	// サブツール
	if len(tool.Subtools) > 0 {
		var subList []map[string]interface{}
		for _, sub := range tool.Subtools {
			name := safeDirName(sub.Name)
			subList = append(subList, map[string]interface{}{"path": name})
			if err := exportSubtool(&sub, filepath.Join(dir, name)); err != nil {
				return err
			}
		}
		meta["tools"] = subList
	}
	return writeMetadata(filepath.Join(dir, "metadata.yaml"), meta)
}

func exportSubtool(sub *Subtool, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	meta := map[string]interface{}{}
	if len(sub.Params) > 0 {
		meta["params"] = sub.Params
	}
	if sub.Script != "" {
		meta["script"] = "main.sh"
		if err := os.WriteFile(filepath.Join(dir, "main.sh"), []byte(sub.Script), 0755); err != nil {
			return err
		}
	}
	if len(sub.BeforeExec) > 0 {
		var beList []map[string]interface{}
		for i, content := range sub.BeforeExec {
			name := "beforeExec_%02d.sh"
			fname := filepath.Join(dir, "beforeExec", fmt.Sprintf(name, i))
			if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fname, []byte(content), 0755); err != nil {
				return err
			}
			beList = append(beList, map[string]interface{}{"path": filepath.Join("beforeExec", fmt.Sprintf(name, i))})
		}
		meta["beforeExec"] = beList
	}
	if len(sub.AfterExec) > 0 {
		var aeList []map[string]interface{}
		for i, content := range sub.AfterExec {
			name := "afterExec_%02d.sh"
			fname := filepath.Join(dir, "afterExec", fmt.Sprintf(name, i))
			if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fname, []byte(content), 0755); err != nil {
				return err
			}
			aeList = append(aeList, map[string]interface{}{"path": filepath.Join("afterExec", fmt.Sprintf(name, i))})
		}
		meta["afterExec"] = aeList
	}
	// ネストしたサブツール
	if len(sub.Subtools) > 0 {
		var subList []map[string]interface{}
		for _, subsub := range sub.Subtools {
			name := safeDirName(subsub.Name)
			subList = append(subList, map[string]interface{}{"path": name})
			if err := exportSubtool(&subsub, filepath.Join(dir, name)); err != nil {
				return err
			}
		}
		meta["tools"] = subList
	}
	return writeMetadata(filepath.Join(dir, "metadata.yaml"), meta)
}

// スペースをアンダースコアに変換するユーティリティ関数
func safeDirName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}
