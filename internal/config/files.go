package config

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	CAFile         = configFile("ca.pem")
	ServerCertFile = configFile("server.pem")
	ServerKeyFile  = configFile("server-key.pem")

	// ClientCertFile = configFile("client.pem")
	// ClientKeyFile  = configFile("client-key.pem")
	RootClientCertFile   = configFile("root-client.pem")
	RootClientKeyFile    = configFile("root-client-key.pem")
	NobodyClientCertFile = configFile("nobody-client.pem")
	NobodyClientKeyFile  = configFile("nobody-client-key.pem")

	ACLModelFile  = configFile("model.conf")
	ACLPolicyFile = configFile("policy.csv")
)

func configFile(filename string) string {
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, filename)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err) // default config dir not found
	}

	filePath := filepath.Join(configDir, "proglog", filename)
	if fileInfo, err := os.Stat(filePath); err != nil {
		panic(err)
	} else if fileInfo.IsDir() {
		panic(fmt.Sprintf("%v is a dir", filename))
	}

	return filePath
}
