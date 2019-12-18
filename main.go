package flokkr

import (
    "errors"
    "fmt"
    "github.com/go-yaml/yaml"
    "github.com/magefile/mage/sh"
    "io/ioutil"
    "net/http"
    "os"
    "path"
    "strings"
)

func GetApacheDownloadUrl(path string) (string, error) {
    url := fmt.Sprintf("https://www-eu.apache.org/dist/" + path)
    resp, err := http.Head(url)
    if err != nil {
        return "", err
    }
    if resp.StatusCode == 200 {
        return url, nil
    }
    url = fmt.Sprintf("https://archive.apache.org/dist/" + path)
    resp, err = http.Head(url)
    if err != nil {
        return "", err
    }
    if resp.StatusCode == 200 {
        return url, nil
    }
    return "", errors.New("Can't find download url for " + path)

}

type FlokkrDescriptor struct {
    Versions []string
    UrlPath  string
    BaseTag  string
    Name     string
    Exclude  []string
}

func ReadDescriptor() (FlokkrDescriptor, error) {
    result := FlokkrDescriptor{}
    content, err := ioutil.ReadFile("flokkr.yaml")
    if err != nil {
        return result, err
    }
    err = yaml.Unmarshal(content, &result)
    if err != nil {
        return result, err
    }
    return result, nil
}

func (desc FlokkrDescriptor) VersionsAndTags() map[string][]string {

    result := make(map[string][]string)
    usedTags := make(map[string]bool)

    first := true
    for _, version := range desc.Versions {
        tags := make([]string, 0)
        if first {
            tags = append(tags, "latest")
            first = false
        }
        parts := strings.Split(version, ".")
        for i := len(parts); i > 0; i-- {
            tag := strings.Join(parts[0:i], ".")
            {
                if _, ok := usedTags[tag]; !ok {
                    tags = append(tags, tag)
                    usedTags[tag] = true
                }
            }

        }
        result[version] = tags
    }
    return result
}

func BuildImage(baseTag string, tag string, dir string, artifact string) error {
    return sh.Run("docker", "build", "-t", tag, "--build-arg", "ARTIFACTDIR="+artifact, "--build-arg", "BASE="+baseTag, dir)
}

func DeployImage(tag string) error {
    return sh.Run("docker", "push", tag)
}

func BuildContainer(desc FlokkrDescriptor, version string, tags []string) error {
    cacheDir, err := downloadIfRequired(desc.UrlPath, version, desc.Name, desc.Exclude)
    if err != nil {
        return err
    }

    err = BuildImage(desc.BaseTag, "flokkr/"+desc.Name+":build", ".", cacheDir)
    if err != nil {
        return err
    }
    for _, tag := range tags {
        err = sh.Run("docker", "tag", "flokkr/"+desc.Name+":build", "flokkr/"+desc.Name+":"+tag)
        if err != nil {
            return err
        }
    }
    return nil
}

func downloadIfRequired(downloadPattern string, version string, name string, excludes []string) (string, error) {
    url, err := GetApacheDownloadUrl(fmt.Sprintf(downloadPattern, version, version))
    if err != nil {
        return "", err
    }
    cacheDir := fmt.Sprintf(".cache/%s/%s", name, version)
    if _, err := os.Stat(cacheDir); err != nil && os.IsNotExist(err) {
        //let me download it
        cacheTmpDir := path.Join(".cache", "work")
        err := os.RemoveAll(cacheTmpDir)
        if err != nil {
            return "", err
        }

        err = os.MkdirAll(cacheTmpDir, 0755)
        if err != nil {
            return "", err
        }
        defer os.RemoveAll(cacheTmpDir)

        downloadedFile := path.Join(cacheTmpDir, "downloaded.tar.gz")
        err = sh.Run("wget", url, "-O", downloadedFile)
        if err != nil {
            return "", err
        }

        err = os.MkdirAll(cacheDir, 0755)
        if err != nil {
            return "", err
        }

        err = sh.Run("tar", "xzf", downloadedFile, "--directory", cacheDir, "--strip-components=1")
        if err != nil {
            return "", err
        }

        for _, exclude := range excludes {
            err := os.RemoveAll(path.Join(cacheDir, exclude))
            if err != nil {
                return "", err
            }

        }
    } else {
        println("artifact is cached locally at " + cacheDir)
    }
    return cacheDir, nil
}

func UpdateBuildBinary() error {
    return sh.Run("mage", "-compile", "./build");
}

func Build() error {
    desc, err := ReadDescriptor()
    if err != nil {
        return err
    }
    for version, tags := range desc.VersionsAndTags() {
        err := BuildContainer(desc, version, tags)
        if err != nil {
            return err
        }
    }
    return nil
}

func Deploy() error {
    desc, err := ReadDescriptor()
    if err != nil {
        return err
    }

    for _, tags := range desc.VersionsAndTags() {
        for _, tag := range tags {
            err = DeployImage("flokkr/" + desc.Name + ":" + tag)
            if err != nil {
                return err
            }
        }
    }
    return nil
}
