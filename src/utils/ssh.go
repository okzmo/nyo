package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
)

func getSSHConfigElements(homeDir string, sshConnection string, configFile *os.File) (string, string, string, []byte, error) {
	cfg, err := ssh_config.Decode(configFile)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to decode ssh config file: %w", err)
	}

	hostname, err := cfg.Get(sshConnection, "HostName")
	if err != nil || hostname == "" {
		return "", "", "", nil, fmt.Errorf("failed to read HostName for Host %s: %w", sshConnection, err)
	}

	port, err := cfg.Get(sshConnection, "Port")
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to read Port for Host %s: %w", sshConnection, err)
	} else if port == "" {
		port = "22"
	}

	user, err := cfg.Get(sshConnection, "User")
	if err != nil || user == "" {
		return "", "", "", nil, fmt.Errorf("failed to read User for Host %s: %w", sshConnection, err)
	}

	identityFile, err := cfg.Get(sshConnection, "IdentityFile")
	if err != nil || identityFile == "" {
		return "", "", "", nil, fmt.Errorf("failed to read IdentityFile for Host %s: %w", sshConnection, err)
	}

	if len(identityFile) > 0 && identityFile[0] == '~' {
		identityFile = filepath.Join(homeDir, identityFile[1:])
	}

	key, err := os.ReadFile(identityFile)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to get key from IdentityFile: %w", err)
	}

	return hostname, port, user, key, nil
}

func ConnectToNode(sshConnection string) (*ssh.Client, string, error) {
	var role string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	sshConfig := filepath.Join(homeDir, ".ssh", "config")
	configFile, err := os.Open(sshConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open ssh config file: %w", err)
	}
	defer configFile.Close()

	hostname, port, user, key, err := getSSHConfigElements(homeDir, sshConnection, configFile)
	if err != nil {
		return nil, "", err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if strings.Contains(err.Error(), "passphrase") {
			fmt.Println("Enter the passphrase for the key: ")
			var passphrase string
			fmt.Scanln(&passphrase)

			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse encrypted private key from IdentityFile: %w", err)
			}
		} else {
			return nil, "", fmt.Errorf("failed to parse private key from IdentityFile: %w", err)
		}
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", hostname, port), config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to dial ssh connection %s: %w", fmt.Sprintf("%s:%s", hostname, port), err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open a session with the ssh client: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("cat /etc/nyo_users")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get users list: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(signer.PublicKey())
	publicKey := strings.TrimSpace(string(publicKeyBytes))
	users := string(output)
	found := false
	for user := range strings.SplitSeq(users, "\n") {
		if user == "" {
			continue
		}

		parts := strings.Split(user, " ")
		if len(parts) >= 3 {
			pubKey := parts[1] + " " + parts[2]

			if strings.Contains(publicKey, pubKey) {
				role = parts[3]
				found = true
				break
			}
		}
	}

	if !found {
		return nil, "", fmt.Errorf("no matching user found in /etc/nyo_users for your SSH key")
	}

	return client, role, nil
}
