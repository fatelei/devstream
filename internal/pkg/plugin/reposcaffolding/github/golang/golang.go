package golang

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/devstream-io/devstream/pkg/util/github"
	"github.com/devstream-io/devstream/pkg/util/log"
)

const (
	DefaultWorkPath      = ".github-repo-scaffolding-golang"
	DefaultTemplateRepo  = "dtm-scaffolding-golang"
	DefaultTemplateOwner = "devstream-io"
	TransitBranch        = "init-with-devstream"
	DefaultMainBranch    = "main"
)

type Config struct {
	AppName   string
	ImageRepo string
	Repo      Repo
}

type Repo struct {
	Name  string
	Owner string
}

func InitRepoLocalAndPushToRemote(repoPath string, opts *Options, ghClient *github.Client) error {
	var retErr error
	// It's ok to give the opts.Org to CreateRepo() when create a repository for a authenticated user.
	if err := ghClient.CreateRepo(opts.Org); err != nil {
		log.Errorf("Failed to create repo: %s.", err)
		return err
	}
	log.Infof("The repo %s has been created.", opts.Repo)

	defer func() {
		if retErr == nil {
			return
		}
		// need to clean the repo created when retErr != nil
		if err := ghClient.DeleteRepo(); err != nil {
			log.Errorf("Failed to delete the repo %s: %s.", opts.Repo, err)
		}
	}()

	if retErr = WalkLocalRepoPath(repoPath, opts, ghClient); retErr != nil {
		log.Debugf("Failed to walk local repo-path: %s.", retErr)
		return retErr
	}

	mainBranch := getMainBranchName(opts)
	if retErr = MergeCommits(ghClient, mainBranch); retErr != nil {
		log.Debugf("Failed to merge commits: %s.", retErr)
		return retErr
	}

	return nil
}

func WalkLocalRepoPath(repoPath string, opts *Options, ghClient *github.Client) error {
	appName := opts.Repo
	mainBranch := getMainBranchName(opts)

	if err := filepath.Walk(repoPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Debugf("Walk error: %s.", err)
			return err
		}

		if info.IsDir() {
			log.Debugf("Found dir: %s.", path)
			return nil
		}
		log.Debugf("Found file: %s.", path)

		if strings.Contains(path, ".git/") {
			log.Debugf("Ignore this file -> %s.", "./git/xxx")
			return nil
		}

		if strings.HasSuffix(path, "README.md") {
			log.Debugf("Ignore this file -> %s.", "README.md")
			return nil
		}

		pathForGithub, err := genPathForGithub(path)
		if err != nil {
			return err
		}

		var content []byte
		if strings.Contains(path, "tpl") {
			content, err = Render(path, opts)
			if err != nil {
				return err
			}
		} else {
			content, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		}
		log.Debugf("Content size: %d", len(content))

		if newPathForGithub, err := replaceAppNameInPathStr(pathForGithub, appName); err != nil {
			return err
		} else {
			// the main branch needs a initial commit
			if strings.Contains(newPathForGithub, "gitignore") {
				err := ghClient.CreateFile(content, strings.TrimSuffix(newPathForGithub, ".tpl"), mainBranch)
				if err != nil {
					log.Debugf("Failed to add the .gitignore file: %s.", err)
					return err
				}
				log.Debugf("Added the .gitignore file.")
				return ghClient.NewBranch(mainBranch, TransitBranch)
			}
			return ghClient.CreateFile(content, strings.TrimSuffix(newPathForGithub, ".tpl"), TransitBranch)
		}
	}); err != nil {
		return err
	}

	return nil
}

func MergeCommits(ghClient *github.Client, mainBranch string) error {
	number, err := ghClient.NewPullRequest(TransitBranch, mainBranch)
	if err != nil {
		return err
	}

	return ghClient.MergePullRequest(number, github.MergeMethodSquash)
}

func Render(filePath string, opts *Options) ([]byte, error) {
	var owner = opts.Owner
	if opts.Org != "" {
		owner = opts.Org
	}

	config := Config{
		AppName:   opts.Repo,
		ImageRepo: opts.ImageRepo,
		Repo: Repo{
			Name:  opts.Repo,
			Owner: owner,
		},
	}
	log.Debugf("FilePath: %s.", filePath)
	log.Debugf("Config %v", config)

	textBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	textStr := string(textBytes)

	tpl := template.New("github-repo-scaffolding-golang").Delims("[[", "]]")
	parsed, err := tpl.Parse(textStr)
	if err != nil {
		log.Debugf("Template parse file failed: %s.", err)
		return nil, err
	}

	var buf bytes.Buffer
	if err = parsed.Execute(&buf, config); err != nil {
		log.Debugf("Template execute failed: %s.", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func genPathForGithub(filePath string) (string, error) {
	splitStrs := strings.SplitN(filePath, "/", 3)
	if len(splitStrs) != 3 {
		return "", fmt.Errorf("unknown format: %s", filePath)
	}
	retStr := splitStrs[2]
	log.Debugf("Path for github: %s.", retStr)
	return retStr, nil
}

func replaceAppNameInPathStr(filePath, appName string) (string, error) {
	log.Debugf("Got filePath %s.", filePath)

	pet := "_app_name_"
	reg, err := regexp.Compile(pet)
	if err != nil {
		return "", err
	}
	newFilePath := reg.ReplaceAllString(filePath, appName)

	log.Debugf("New filePath: \"%s\".", newFilePath)

	return newFilePath, nil
}

func buildState(opts *Options) map[string]interface{} {
	res := make(map[string]interface{})
	res["owner"] = opts.Owner
	res["org"] = opts.Org
	res["repoName"] = opts.Repo

	outputs := make(map[string]interface{})
	outputs["owner"] = opts.Owner
	outputs["org"] = opts.Org
	outputs["repo"] = opts.Repo
	if opts.Owner != "" {
		outputs["repoURL"] = fmt.Sprintf("https://github.com/%s/%s.git", opts.Owner, opts.Repo)
	} else {
		outputs["repoURL"] = fmt.Sprintf("https://github.com/%s/%s.git", opts.Org, opts.Repo)
	}
	res["outputs"] = outputs

	return res
}

func getMainBranchName(opts *Options) string {
	if opts.Branch == "" {
		return DefaultMainBranch
	}
	return opts.Branch
}
