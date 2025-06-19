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
	fmt.Printf("\n[DEBUG] Starting Compile method for root dir: %s\n", b.RootDir)
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
		
		processedDirs := make(map[string]bool)
		
		rootDirAbs, err := filepath.Abs(b.RootDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path of root directory %s: %w", b.RootDir, err)
		}
		processedDirs[rootDirAbs] = true
		
		for _, t := range toolDefs {
			if path, ok := t["path"].(string); ok {
				if path == "." {
					
					fmt.Printf("\n[DEBUG] Found path: . in directory: %s\n", b.RootDir)
					entries, err := os.ReadDir(b.RootDir)
					if err != nil {
						return nil, fmt.Errorf("failed to read directory %s: %w", b.RootDir, err)
					}
					
					for _, entry := range entries {
						if !entry.IsDir() {
							continue
						}
						
						subDir := filepath.Join(b.RootDir, entry.Name())
						subDirAbs, err := filepath.Abs(subDir)
						if err != nil {
							return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", subDir, err)
						}
						
						if processedDirs[subDirAbs] {
							fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", subDirAbs)
							continue
						}
						
						metadataPath := filepath.Join(subDir, "metadata.yaml")
						if _, err := os.Stat(metadataPath); err == nil {
							tool, err := buildTool(subDir, processedDirs)
							if err != nil {
								return nil, err
							}
							cfg.Tools = append(cfg.Tools, *tool)
							processedDirs[subDirAbs] = true
						}
					}
					continue
				}
				
				fullPath := filepath.Join(b.RootDir, path)
				fullPathAbs, err := filepath.Abs(fullPath)
				if err != nil {
					return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", fullPath, err)
				}
				
				if processedDirs[fullPathAbs] {
					fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", fullPathAbs)
					continue
				}
				
				fileInfo, err := os.Stat(fullPath)
				if err != nil {
					return nil, fmt.Errorf("failed to stat path %s: %w", fullPath, err)
				}
				
				metadataPath := filepath.Join(fullPath, "metadata.yaml")
				_, metadataErr := os.Stat(metadataPath)
				
				if fileInfo.IsDir() && metadataErr == nil {
					tool, err := buildTool(fullPath, processedDirs)
					if err != nil {
						return nil, err
					}
					cfg.Tools = append(cfg.Tools, *tool)
					processedDirs[fullPathAbs] = true
				} else if fileInfo.IsDir() {
					subDirs, err := findToolDirectories(fullPath, processedDirs)
					if err != nil {
						return nil, err
					}
					
					for _, subDir := range subDirs {
						tool, err := buildTool(subDir, processedDirs)
						if err != nil {
							return nil, err
						}
						cfg.Tools = append(cfg.Tools, *tool)
					}
				} else {
					tool, err := buildTool(fullPath, processedDirs)
					if err != nil {
						return nil, err
					}
					cfg.Tools = append(cfg.Tools, *tool)
				}
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

func findToolDirectories(dir string, processedDirs map[string]bool) ([]string, error) {
	fmt.Printf("\n[DEBUG] findToolDirectories: %s\n", dir)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", dir, err)
	}
	
	if processedDirs[absDir] {
		fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", absDir)
		fmt.Printf("\n[DEBUG] Already processed directory: %s\n", absDir)
		return []string{}, nil
	}
	
	processedDirs[absDir] = true
	fmt.Printf("\n[DEBUG] Marking directory as processed: %s\n", absDir)
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	
	var toolDirs []string
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		subDir := filepath.Join(dir, entry.Name())
		subDirAbs, err := filepath.Abs(subDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", subDir, err)
		}
		
		if processedDirs[subDirAbs] {
						fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", subDirAbs)
			continue
		}
		
		metadataPath := filepath.Join(subDir, "metadata.yaml")
		
		if _, err := os.Stat(metadataPath); err == nil {
			toolDirs = append(toolDirs, subDir)
			processedDirs[subDirAbs] = true
		}
	}
	
	return toolDirs, nil
}

// buildTool はツールディレクトリからTool構造体を構築します。
func buildTool(dir string, processedDirs map[string]bool) (*Tool, error) {
	fmt.Printf("\n[DEBUG] buildTool: %s\n", dir)
	meta, err := readMetadata(filepath.Join(dir, "metadata.yaml"))
	if err != nil {
		return nil, err
	}
	
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", dir, err)
	}
	
	if processedDirs[absDir] && dir != absDir {
		return nil, fmt.Errorf("directory %s is already processed", dir)
	}
	
	processedDirs[absDir] = true
	fmt.Printf("\n[DEBUG] Marking directory as processed: %s\n", absDir)
	
	tool := &Tool{}
	tool.Name = filepath.Base(dir)
	if description, ok := meta["description"].(string); ok {
		tool.Description = description
	}
	if params, ok := meta["params"]; ok {
		paramsYaml, _ := yaml.Marshal(params)
		if err := yaml.Unmarshal(paramsYaml, &tool.Params); err != nil {
			return nil, err
		}
	}
	if script, ok := meta["script"].(string); ok {
		scriptPath := filepath.Join(dir, script)
		if content, err := os.ReadFile(scriptPath); err == nil {
			tool.Script = string(content)
		} else {
			tool.Script = script
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
	// サブツール
	if subs, ok := meta["tools"]; ok {
		var subDefs []map[string]interface{}
		subsYaml, _ := yaml.Marshal(subs)
		if err := yaml.Unmarshal(subsYaml, &subDefs); err != nil {
			return nil, err
		}
		
		for _, s := range subDefs {
			if path, ok := s["path"].(string); ok {
				if path == "." {
					fmt.Printf("\n[DEBUG] Found path: . in directory: %s\n", dir)
					continue
				}
				
				subDir := filepath.Join(dir, path)
				absSubDir, err := filepath.Abs(subDir)
				if err != nil {
					return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", subDir, err)
				}
				
				if processedDirs[absSubDir] {
					fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", absSubDir)
					continue
				}
				
				// サブツールのmetadata.yamlを読み込む
				subMeta, err := readMetadata(filepath.Join(subDir, "metadata.yaml"))
				if err != nil {
					return nil, err
				}
				// 親ツールのパラメータ情報を追加
				subMeta["parent_params"] = tool.Params
				// サブツールをビルド
				sub, err := buildSubtool(subDir, processedDirs)
				if err != nil {
					return nil, err
				}
				tool.Subtools = append(tool.Subtools, *sub)
				processedDirs[absSubDir] = true
			}
		}
	}
	
	return tool, nil
}

// buildSubtool はサブツールディレクトリからTool構造体を構築します。
func buildSubtool(dir string, processedDirs map[string]bool) (*Tool, error) {
	fmt.Printf("\n[DEBUG] buildSubtool: %s\n", dir)
	meta, err := readMetadata(filepath.Join(dir, "metadata.yaml"))
	if err != nil {
		return nil, err
	}
	
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", dir, err)
	}
	
	if processedDirs[absDir] && dir != absDir {
		return nil, fmt.Errorf("directory %s is already processed", dir)
	}
	
	processedDirs[absDir] = true
	fmt.Printf("\n[DEBUG] Marking directory as processed: %s\n", absDir)
	
	sub := &Tool{}
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
		paramRefsMap := make(map[string]Parameter)
		paramRefsYaml, _ := yaml.Marshal(paramRefs)
		var tempMap map[string]map[string]interface{}
		if err := yaml.Unmarshal(paramRefsYaml, &tempMap); err != nil {
			return nil, err
		}
		for name, paramRef := range tempMap {
			param := Parameter{}
			if required, ok := paramRef["required"].(bool); ok {
				param.Required = required
			}
			// 親ツールのパラメータ情報を取得
			if parentParams, ok := meta["parent_params"].(map[string]interface{}); ok {
				if parentParam, ok := parentParams[name].(map[string]interface{}); ok {
					if desc, ok := parentParam["description"].(string); ok {
						param.Description = desc
					}
					if typ, ok := parentParam["type"].(string); ok {
						param.Type = typ
					}
				}
			}
			paramRefsMap[name] = param
		}
		if sub.Params == nil {
			sub.Params = make(map[string]Parameter)
		}
		for name, param := range paramRefsMap {
			sub.Params[name] = param
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
				if path == "." {
					fmt.Printf("\n[DEBUG] Found path: . in directory: %s\n", dir)
					continue
				}
				
				subDir := filepath.Join(dir, path)
				absSubDir, err := filepath.Abs(subDir)
				if err != nil {
					return nil, fmt.Errorf("failed to get absolute path of directory %s: %w", subDir, err)
				}
				
				if processedDirs[absSubDir] {
					fmt.Printf("\n[DEBUG] Directory already processed, skipping: %s\n", absSubDir)
					continue
				}
				
				subsub, err := buildSubtool(subDir, processedDirs)
				if err != nil {
					return nil, err
				}
				sub.Subtools = append(sub.Subtools, *subsub)
				processedDirs[absSubDir] = true
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

func WriteMetadata(path string, meta map[string]interface{}) error {
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

func writeMetadata(path string, meta map[string]interface{}) error {
	return WriteMetadata(path, meta)
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

func exportSubtool(sub *Tool, dir string) error {
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

