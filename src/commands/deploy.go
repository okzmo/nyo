package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/okzmo/nyo/src/utils"
	"github.com/pelletier/go-toml/v2"
)

var reservedKeywords = []string{"database"}

type ServiceConfig struct {
	Name    string
	Domain  string   `toml:"domain"`
	Spa     bool     `toml:"spa"`
	Path    string   `toml:"path"`
	Use     string   `toml:"use"`
	Prepare []string `toml:"prepare"`
	Nodes   []string `toml:"nodes"`
	Tools   []string `toml:"tools"`
}

type DatabaseConfig struct {
	Type     string `toml:"type"`
	Name     string `toml:"name"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

func checkNyoFile(workDir string) (string, error) {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to read the current dir: %w", err)
	}

	found := false
	for _, entry := range entries {
		if entry.Name() == "Nyo.toml" {
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("missing Nyo.toml file in the current directory")
	}

	return filepath.Join(workDir, "Nyo.toml"), nil
}

func buildDatabaseConfig(sectionData map[string]any) (DatabaseConfig, error) {
	var dbConfig DatabaseConfig

	if val, ok := sectionData["type"]; !ok {
		return DatabaseConfig{}, fmt.Errorf("failed to get the type of the database service")
	} else {
		dbConfig.Type = val.(string)
	}

	if val, ok := sectionData["name"]; !ok {
		return DatabaseConfig{}, fmt.Errorf("failed to get the name of the database")
	} else {
		dbConfig.Name = val.(string)
	}

	if val, ok := sectionData["username"]; !ok {
		return DatabaseConfig{}, fmt.Errorf("failed to get the username for the database")
	} else {
		dbConfig.Username = val.(string)
	}

	if val, ok := sectionData["password"]; !ok {
		return DatabaseConfig{}, fmt.Errorf("failed to get the password for the database")
	} else {
		dbConfig.Password = val.(string)
	}

	return dbConfig, nil
}

func buildServiceConfig(sectionData map[string]any) (ServiceConfig, error) {
	var serviceConfig ServiceConfig

	if val, ok := sectionData["domain"]; ok {
		serviceConfig.Domain = val.(string)
	}

	if val, ok := sectionData["spa"]; ok {
		serviceConfig.Spa = val.(bool)
	}

	if val, ok := sectionData["path"]; !ok {
		return ServiceConfig{}, fmt.Errorf("missing required 'path' field")
	} else {
		serviceConfig.Path = val.(string)
	}

	if val, ok := sectionData["use"]; !ok {
		return ServiceConfig{}, fmt.Errorf("missing required 'use' field")
	} else {
		serviceConfig.Use = val.(string)
	}

	if val, ok := sectionData["prepare"]; ok {
		prepareSlice, err := utils.ConvertToStringSlice(val)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid prepare commands: %w", err)
		}
		serviceConfig.Prepare = prepareSlice
	} else {
		return ServiceConfig{}, fmt.Errorf("missing required 'prepare' field")
	}

	if val, ok := sectionData["nodes"]; ok {
		nodesSlice, err := utils.ConvertToStringSlice(val)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid nodes list: %w", err)
		}
		serviceConfig.Nodes = nodesSlice
	} else {
		return ServiceConfig{}, fmt.Errorf("missing required 'nodes' field")
	}

	if val, ok := sectionData["tools"]; ok {
		toolsSlice, err := utils.ConvertToStringSlice(val)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid tools list: %w", err)
		}
		serviceConfig.Tools = toolsSlice
	} else {
		return ServiceConfig{}, fmt.Errorf("missing required 'tools' field")
	}

	return serviceConfig, nil
}

func parseNyoFile(path string) ([]ServiceConfig, []DatabaseConfig, error) {
	var projectName string
	var dbsConfig []DatabaseConfig
	var servicesConfig []ServiceConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the Nyo.toml file: %w", err)
	}

	var config map[string]any
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Nyo.toml: %w", err)
	}

	name, ok := config["name"]
	if !ok {
		return nil, nil, fmt.Errorf("missing project name in your Nyo.toml file, please add a name at the beginning. (e.g. name = 'example')")
	}
	projectName = name.(string)

	for sectionName, sectionData := range config {
		if sectionName == "name" {
			continue
		}

		if kw, ok := utils.ContainsSubstring(sectionName, reservedKeywords); ok {
			if kw == "database" {
				dbConfig, err := buildDatabaseConfig(sectionData.(map[string]any))
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse database config: %w", err)
				}
				dbsConfig = append(dbsConfig, dbConfig)
			}
		} else {
			var serviceConfig ServiceConfig

			serviceConfig, err := buildServiceConfig(sectionData.(map[string]any))
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse service config: %w", err)
			}
			serviceConfig.Name = projectName + "-" + sectionName
			servicesConfig = append(servicesConfig, serviceConfig)
		}
	}

	return servicesConfig, dbsConfig, nil
}

func Deploy() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the current working dir: %w", err)
	}

	path, err := checkNyoFile(dir)
	if err != nil {
		return err
	}

	services, dbs, err := parseNyoFile(path)
	if err != nil {
		return err
	}

	fmt.Println(services, dbs)

	for _, service := range services {
		for _, node := range service.Nodes {
			client, _, err := utils.ConnectToNode(node)
			if err != nil {
				return fmt.Errorf("failed to connect to the given node %s: %w", node, err)
			}
			defer client.Close()

			session, err := client.NewSession()
			if err != nil {
				return fmt.Errorf("failed to open a session with the ssh client: %w", err)
			}
			defer session.Close()

			output, err := session.CombinedOutput("cat /etc/nyo_users")
			if err != nil {
				return fmt.Errorf("failed to get users list: %w", err)
			}
			fmt.Println(output)
		}
	}

	return nil
}
