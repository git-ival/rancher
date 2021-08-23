package installer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/rancher/pkg/systemtemplate"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	SystemAgentInstallPath = "/system-agent-install.sh" // corresponding curl -o in package/Dockerfile
	WindowsRke2InstallPath = "/rke2-agent-install.ps1"  // corresponding curl -o in package/Dockerfile
)

var (
	localAgentInstallScripts = []string{
		"/usr/share/rancher/ui/assets" + SystemAgentInstallPath,
		"." + SystemAgentInstallPath,
	}
	localWindowsRke2InstallScripts = []string{
		"./rke2-install.ps1",
	}
)

func installScript(setting settings.Setting, files []string) ([]byte, error) {
	if setting.Get() == setting.Default {
		// no setting override, check for local file first
		for _, f := range files {
			script, err := ioutil.ReadFile(f)
			if err != nil {
				if !os.IsNotExist(err) {
					logrus.Debugf("error pulling system agent installation script %s: %s", f, err)
				}
				continue
			}
			return script, err
		}
		logrus.Debugf("no local installation script found, moving on to url: %s", setting.Get())
	}

	resp, err := http.Get(setting.Get())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func LinuxInstallScript(token string, envVars []corev1.EnvVar, defaultHost string) ([]byte, error) {
	data, err := installScript(
		settings.SystemAgentInstallScript,
		localAgentInstallScripts)
	if err != nil {
		return nil, err
	}
	binaryURL := ""
	if settings.SystemAgentVersion.Get() != "" {
		if settings.ServerURL.Get() != "" {
			binaryURL = fmt.Sprintf("CATTLE_AGENT_BINARY_BASE_URL=\"%s/assets\"", settings.ServerURL.Get())
		} else if defaultHost != "" {
			binaryURL = fmt.Sprintf("CATTLE_AGENT_BINARY_BASE_URL=\"https://%s/assets\"", defaultHost)
		}
	}
	ca := systemtemplate.CAChecksum()
	if ca != "" {
		ca = "CATTLE_CA_CHECKSUM=\"" + ca + "\""
	}
	if token != "" {
		token = "CATTLE_ROLE_NONE=true\nCATTLE_TOKEN=\"" + token + "\""
	}
	envVarBuf := &strings.Builder{}
	for _, envVar := range envVars {
		if envVar.Value == "" {
			continue
		}
		envVarBuf.WriteString(fmt.Sprintf("%s=\"%s\"\n", envVar.Name, envVar.Value))
	}
	server := ""
	if settings.ServerURL.Get() != "" {
		server = fmt.Sprintf("CATTLE_SERVER=%s", settings.ServerURL.Get())
	}
	return []byte(fmt.Sprintf(`#!/usr/bin/env sh
%s
%s
%s
%s
%s

%s
`, envVarBuf.String(), binaryURL, server, ca, token, data)), nil
}

func WindowsInstallScript(token string, envVars []corev1.EnvVar, defaultHost string) ([]byte, error) {
	data, err := installScript(
		settings.WindowsRke2InstallScript,
		localWindowsRke2InstallScripts)
	if err != nil {
		return nil, err
	}

	ca := systemtemplate.CAChecksum()
	if ca != "" {
		ca = "$env:CATTLE_CA_CHECKSUM=\"" + ca + "\""
	}
	if token != "" {
		token = "$env:CATTLE_ROLE_NONE=true\n$env:CATTLE_TOKEN=\"" + token + "\""
	}
	envVarBuf := &strings.Builder{}
	for _, envVar := range envVars {
		if envVar.Value == "" {
			continue
		}
		envVarBuf.WriteString(fmt.Sprintf("$env:%s=\"%s\"\n", envVar.Name, envVar.Value))
	}
	server := ""
	if settings.ServerURL.Get() != "" {
		server = fmt.Sprintf("$env:CATTLE_SERVER=\"%s\"", settings.ServerURL.Get())
	}

	return []byte(fmt.Sprintf(`%s

%s
%s
%s
%s

Rke2-Installer @PSBoundParameters
exit 0
`, data, envVarBuf.String(), server, ca, token)), nil
}
