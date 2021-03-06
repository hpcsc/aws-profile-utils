package checker

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func githubStubResponsePath(url string) string {
	if strings.Contains(url, "releases/latest") {
		return "testdata/github-latest-release-response.json"
	}

	return "testdata/github-get-commit-response.json"
}

func readFile(path string) ([]byte, error) {
	response, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("failed to read response file: %v", err)
		os.Exit(1)
	}

	return response, nil
}

var stubGithubGetUrl = func(url string, authorization string) ([]byte, error) {
	return readFile(githubStubResponsePath(url))
}

func TestGithubChecker_LatestVersionUrl(t *testing.T) {
	var testCases = []struct {
		os              string
		expectedUrl     string
		expectedVersion string
	}{
		{"windows", "https://github.com/hpcsc/aws-profile/releases/download/v0.4.0/aws-profile-windows.exe", "ccc4227f44a69597aac7cd9fa516132fb37dacca"},
		{"linux", "https://github.com/hpcsc/aws-profile/releases/download/v0.4.0/aws-profile-linux", "ccc4227f44a69597aac7cd9fa516132fb37dacca"},
		{"macos", "https://github.com/hpcsc/aws-profile/releases/download/v0.4.0/aws-profile-macos", "ccc4227f44a69597aac7cd9fa516132fb37dacca"},
	}

	for _, tt := range testCases {
		t.Run(fmt.Sprintf("return version link to latest %s binary when os is %s", tt.os, tt.os), func(t *testing.T) {
			c := newGithubChecker(tt.os, stubGithubGetUrl)

			url, version, err := c.LatestVersionUrl()

			require.NoError(t, err)
			require.Equal(t, tt.expectedUrl, url)
			require.Equal(t, tt.expectedVersion, version)
		})
	}

	t.Run("access github with authorization token if available in environment variable", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "some-token")

		var calledAuthorization string
		c := newGithubChecker("linux", func(url string, authorization string) ([]byte, error) {
			calledAuthorization = authorization
			return readFile(githubStubResponsePath(url))
		})

		_, _, err := c.LatestVersionUrl()

		require.NoError(t, err)
		require.Equal(t, "some-token", calledAuthorization)

		os.Unsetenv("GITHUB_TOKEN")
	})

	t.Run("access github with no authorization token if environment variable is not set", func(t *testing.T) {
		var calledAuthorization string
		c := newGithubChecker("linux", func(url string, authorization string) ([]byte, error) {
			calledAuthorization = authorization
			return readFile(githubStubResponsePath(url))
		})

		_, _, err := c.LatestVersionUrl()

		require.NoError(t, err)
		require.Empty(t, calledAuthorization)
	})

	t.Run("return link even if not able to get commit hash for latest version", func(t *testing.T) {
		c := newGithubChecker("linux", func(url string, authorization string) ([]byte, error) {
			if strings.Contains(url, "releases/latest") {
				return readFile("testdata/github-latest-release-response.json")
			}

			return nil, errors.New("some error")
		})

		url, version, err := c.LatestVersionUrl()

		require.NoError(t, err)
		require.Equal(t, "https://github.com/hpcsc/aws-profile/releases/download/v0.4.0/aws-profile-linux", url)
		require.Equal(t, "", version)
	})

	t.Run("return error when fail to get url", func(t *testing.T) {
		c := newGithubChecker("linux", func(url string, authorization string) ([]byte, error) {
			return nil, errors.New("some error")
		})

		_, _, err := c.LatestVersionUrl()

		require.Error(t, err)
		require.Contains(t, err.Error(), "some error")
	})

	t.Run("return error when fail to unmarshal get latest response", func(t *testing.T) {
		c := newGithubChecker("linux", func(url string, authorization string) ([]byte, error) {
			return []byte("{ \"Assets\": \"some-assets\"}"), nil
		})

		_, _, err := c.LatestVersionUrl()

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal get latest release response body")
	})

	t.Run("return error when no asset for given os found", func(t *testing.T) {
		c := newGithubChecker("bsd", stubGithubGetUrl)

		_, _, err := c.LatestVersionUrl()

		require.Error(t, err)
		require.Contains(t, err.Error(), "download url for os bsd not found")
	})
}
