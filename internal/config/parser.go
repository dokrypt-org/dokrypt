package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	envVarPattern      = regexp.MustCompile(`\$\{([^}]+)\}`)
	templateVarPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)
)

func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	content := interpolateEnvVars(string(data))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	ApplyDefaults(cfg)

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	overridePath := filepath.Join(dir, name+".override"+ext)

	if _, err := os.Stat(overridePath); err == nil {
		overrideCfg, err := parseRaw(overridePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse override file %s: %w", overridePath, err)
		}
		cfg = MergeConfigs(cfg, overrideCfg)
	}

	return cfg, nil
}

func ParseWithProfile(path, profile string) (*Config, error) {
	cfg, err := Parse(path)
	if err != nil {
		return nil, err
	}

	if profile == "" {
		return cfg, nil
	}

	p, ok := cfg.Profiles[profile]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in config", profile)
	}

	mergeSettings(&cfg.Settings, &p.Settings)
	return cfg, nil
}

func parseRaw(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := interpolateEnvVars(string(data))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func interpolateEnvVars(content string) string {
	return envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[2 : len(match)-1] // strip ${ and }
		name, defaultVal, hasDefault := strings.Cut(inner, ":-")

		val := os.Getenv(name)
		if val == "" && hasDefault {
			return defaultVal
		}
		if val == "" {
			return match // Leave unresolved if no default
		}
		return val
	})
}

func ResolveTemplateVars(cfg *Config, resolver TemplateResolver) error {
	for name, svc := range cfg.Services {
		for key, val := range svc.Environment {
			resolved, err := resolveTemplateString(val, resolver)
			if err != nil {
				return fmt.Errorf("service %s, env %s: %w", name, key, err)
			}
			svc.Environment[key] = resolved
		}
		cfg.Services[name] = svc
	}
	return nil
}

type TemplateResolver interface {
	ResolveChain(chainName, field string) (string, error)
	ResolveService(serviceName, field string) (string, error)
	ResolveDeploy(label, field string) (string, error)
}

func resolveTemplateString(s string, resolver TemplateResolver) (string, error) {
	return replaceAllStringSubmatchFunc(templateVarPattern, s, func(groups []string) (string, error) {
		if len(groups) < 2 {
			return groups[0], nil
		}

		ref := strings.TrimSpace(groups[1])
		parts := strings.SplitN(ref, ".", 3)
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid template variable %q: expected category.name.field", ref)
		}

		category, name, field := parts[0], parts[1], parts[2]
		switch category {
		case "chain":
			return resolver.ResolveChain(name, field)
		case "service":
			return resolver.ResolveService(name, field)
		case "deploy":
			return resolver.ResolveDeploy(name, field)
		default:
			return "", fmt.Errorf("unknown template variable category %q", category)
		}
	})
}

func replaceAllStringSubmatchFunc(re *regexp.Regexp, s string, fn func([]string) (string, error)) (string, error) {
	var result strings.Builder
	lastIndex := 0

	for _, match := range re.FindAllStringSubmatchIndex(s, -1) {
		result.WriteString(s[lastIndex:match[0]])

		groups := make([]string, 0)
		for i := 0; i < len(match); i += 2 {
			if match[i] >= 0 {
				groups = append(groups, s[match[i]:match[i+1]])
			} else {
				groups = append(groups, "")
			}
		}

		replacement, err := fn(groups)
		if err != nil {
			return "", err
		}
		result.WriteString(replacement)
		lastIndex = match[1]
	}

	result.WriteString(s[lastIndex:])
	return result.String(), nil
}
