package handlers

import (
	"fmt"
	"github.com/hpcsc/aws-profile/internal/io"
	"github.com/hpcsc/aws-profile/internal/log"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"regexp"
	"strings"
)

type ReadCachedCallerIdentityFn func() (string, error)
type WriteCachedCallerIdentityFn func(string) error
type GetAWSCallerIdentityFn func() (string, error)

type GetHandler struct {
	SubCommand                  *kingpin.CmdClause
	GetAWSCallerIdentityFn      GetAWSCallerIdentityFn
	ReadCachedCallerIdentityFn  ReadCachedCallerIdentityFn
	WriteCachedCallerIdentityFn WriteCachedCallerIdentityFn
	Logger                      log.Logger
}

func NewGetHandler(
	app *kingpin.Application,
	logger log.Logger,
	getAWSCallerIdentityFn GetAWSCallerIdentityFn,
	readCachedCallerIdentityFn ReadCachedCallerIdentityFn,
	writeCachedCallerIdentityFn WriteCachedCallerIdentityFn,
) GetHandler {
	subCommand := app.Command("get", "get current AWS profile")

	return GetHandler{
		SubCommand:                  subCommand,
		GetAWSCallerIdentityFn:      getAWSCallerIdentityFn,
		ReadCachedCallerIdentityFn:  readCachedCallerIdentityFn,
		WriteCachedCallerIdentityFn: writeCachedCallerIdentityFn,
		Logger:                      logger,
	}
}

func awsCredentialsEnvironmentVariablesSet() bool {
	var _, accessKeyIdExists = os.LookupEnv("AWS_ACCESS_KEY_ID")
	var _, secretAccessKeyExists = os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	var _, sessionTokenExists = os.LookupEnv("AWS_SESSION_TOKEN")

	return accessKeyIdExists && secretAccessKeyExists && sessionTokenExists
}

func (handler GetHandler) Handle(globalArguments GlobalArguments) (bool, string) {
	if awsCredentialsEnvironmentVariablesSet() {
		cachedCallerIdentity, readCachedCallerIdentityErr := handler.ReadCachedCallerIdentityFn()
		if readCachedCallerIdentityErr == nil && cachedCallerIdentity != "" {
			return true, cachedCallerIdentity
		}

		var callerIdentityProfile, getCallerIdentityErr = handler.GetAWSCallerIdentityFn()
		if getCallerIdentityErr != nil {
			if strings.Contains(getCallerIdentityErr.Error(), "ExpiredToken") {
				return true, "error: ExpiredToken"
			}

			// error returned by aws sometimes has format: "xxx (ErrorCode) yyy"
			// try to parse and get that "ErrorCode" from error first
			// if not possible to parse, return unknown to be printed out
			errorRegex := regexp.MustCompile(`(\(.*?\))`)
			errorMatch := errorRegex.FindStringSubmatch(getCallerIdentityErr.Error())
			if len(errorMatch) < 2 {
				handler.Logger.Errorf("failed to get caller identity with error: %s", getCallerIdentityErr.Error())
				return true, "unknown"
			}

			return true, fmt.Sprintf("error: %s", strings.Trim(errorMatch[1], "()"))
		}

		writeError := handler.WriteCachedCallerIdentityFn(callerIdentityProfile)
		if writeError != nil {
			handler.Logger.Errorf("failed to write caller identity [%s] to cached file", callerIdentityProfile)
		}
		return true, callerIdentityProfile
	} else {
		writeError := handler.WriteCachedCallerIdentityFn("")
		if writeError != nil {
			handler.Logger.Errorf("failed to reset caller identity in cached file")
		}
	}

	configFile, err := io.ReadFile(globalArguments.ConfigFilePath)
	if err != nil {
		return false, fmt.Sprintf("Fail to read AWS config file: %v", err)
	}

	configDefaultSection, err := configFile.GetSection("default")
	if err == nil &&
		configDefaultSection.HasKey("role_arn") &&
		configDefaultSection.HasKey("source_profile") {

		defaultRoleArn := configDefaultSection.Key("role_arn").Value()
		defaultSourceProfile := configDefaultSection.Key("source_profile").Value()

		for _, section := range configFile.Sections() {
			if strings.Compare(section.Name(), "default") != 0 &&
				section.HasKey("role_arn") &&
				section.HasKey("source_profile") &&
				strings.Compare(section.Key("role_arn").Value(), defaultRoleArn) == 0 &&
				strings.Compare(section.Key("source_profile").Value(), defaultSourceProfile) == 0 {
				return true, section.Name()
			}
		}
	}

	credentialsFile, err := io.ReadFile(globalArguments.CredentialsFilePath)
	if err != nil {
		return false, fmt.Sprintf("Fail to read AWS credentials file: %v", err)
	}

	credentialsDefaultSection, err := credentialsFile.GetSection("default")
	if err == nil &&
		credentialsDefaultSection.HasKey("aws_access_key_id") {

		defaultAWSAccessKeyId := credentialsDefaultSection.Key("aws_access_key_id").Value()

		for _, section := range credentialsFile.Sections() {
			if strings.Compare(section.Name(), "default") != 0 &&
				section.HasKey("aws_access_key_id") &&
				strings.Compare(section.Key("aws_access_key_id").Value(), defaultAWSAccessKeyId) == 0 {
				return true, fmt.Sprintf("%s\n", section.Name())
			}
		}
	}

	return true, ""
}
