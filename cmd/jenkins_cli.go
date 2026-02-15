package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

type JenkinsConfig struct {
	JenkinsURL string `json:"jenkins_url"`
	Username   string `json:"username"`
	Secret     string `json:"secret"` // password or API token
}

type crumbResponse struct {
	Crumb             string `json:"crumb"`
	CrumbRequestField string `json:"crumbRequestField"`
}

type buildInfo struct {
	Number    int64  `json:"number"`
	Result    string `json:"result"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"`
	Building  bool   `json:"building"`
}

type jobInfo struct {
	Builds []buildInfo `json:"builds"`
}

func jenkinsConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".jenkins-cli"), nil
}

func loadJenkinsConfig() (*JenkinsConfig, error) {
	dir, err := jenkinsConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var cfg JenkinsConfig
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveJenkinsConfig(cfg *JenkinsConfig) error {
	dir, err := jenkinsConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func askForJenkinsConfig() (*JenkinsConfig, error) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("\nJenkins CLI setup — let’s connect to your server."))

	urlPrompt := promptui.Prompt{Label: "Jenkins URL (e.g. https://jenkins.example.com)", Validate: func(s string) error {
		if s == "" {
			return fmt.Errorf("URL is required")
		}
		if !(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
			return fmt.Errorf("must start with http:// or https://")
		}
		if _, err := url.ParseRequestURI(s); err != nil {
			return err
		}
		return nil
	}}
	jURL, err := urlPrompt.Run()
	if err != nil {
		return nil, err
	}

	userPrompt := promptui.Prompt{Label: "Username", Validate: func(s string) error {
		if s == "" {
			return fmt.Errorf("username is required")
		}
		return nil
	}}
	user, err := userPrompt.Run()
	if err != nil {
		return nil, err
	}

	secretPrompt := promptui.Prompt{Label: "API Token or Password", Mask: '*', Validate: func(s string) error {
		if s == "" {
			return fmt.Errorf("secret cannot be empty")
		}
		return nil
	}}
	secret, err := secretPrompt.Run()
	if err != nil {
		return nil, err
	}

	cfg := &JenkinsConfig{JenkinsURL: strings.TrimRight(jURL, "/"), Username: user, Secret: secret}
	if err := saveJenkinsConfig(cfg); err != nil {
		return nil, err
	}
	color.New(color.FgGreen).Println("Saved configuration to ~/.jenkins-cli/config.json")
	return cfg, nil
}

func basicAuthHeader(user, pass string) string {
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + token
}

func getCrumb(client *http.Client, cfg *JenkinsConfig) (field, crumb string, err error) {
	req, err := http.NewRequest("GET", cfg.JenkinsURL+"/crumbIssuer/api/json", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", basicAuthHeader(cfg.Username, cfg.Secret))
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		// Crumb disabled
		return "", "", nil
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("crumb request failed: %s: %s", resp.Status, string(b))
	}
	var cr crumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", "", err
	}
	return cr.CrumbRequestField, cr.Crumb, nil
}

func triggerBuild(cfg *JenkinsConfig, job string, params map[string]string) error {
	// Use a client with cookie jar so the crumb session is preserved
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 30 * time.Second, Jar: jar}
	field, crumb, _ := getCrumb(client, cfg)

	path := buildJobPathBase(cfg, job)
	var endpoint string
	var body io.Reader
	if len(params) > 0 {
		endpoint = path + "/buildWithParameters"
		form := url.Values{}
		for k, v := range params {
			form.Set(k, v)
		}
		body = strings.NewReader(form.Encode())
	} else {
		endpoint = path + "/build"
		body = nil
	}
	makeRequest := func(addCrumb bool) (*http.Response, error) {
		req, err := http.NewRequest("POST", endpoint, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", basicAuthHeader(cfg.Username, cfg.Secret))
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		// Always set a content length, even for nil body
		if body == nil {
			req.ContentLength = 0
		}
		if addCrumb && field != "" && crumb != "" {
			req.Header.Set(field, crumb)
		}
		return client.Do(req)
	}

	// First attempt with crumb if available
	resp, err := makeRequest(true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 201 || resp.StatusCode == 202 {
		return nil
	}

	// If forbidden due to crumb, retry by refetching crumb once
	if resp.StatusCode == 403 {
		// Read body to check message
		b1, _ := io.ReadAll(resp.Body)
		if strings.Contains(strings.ToLower(string(b1)), "crumb") {
			// Refresh crumb
			if f2, c2, err2 := getCrumb(client, cfg); err2 == nil {
				field, crumb = f2, c2
				// Rebuild body reader since it was consumed
				if len(params) > 0 {
					form := url.Values{}
					for k, v := range params {
						form.Set(k, v)
					}
					body = strings.NewReader(form.Encode())
				} else {
					body = nil
				}
				resp2, err2 := makeRequest(true)
				if err2 != nil {
					return err2
				}
				defer resp2.Body.Close()
				if resp2.StatusCode == 201 || resp2.StatusCode == 202 {
					return nil
				}
				b2, _ := io.ReadAll(resp2.Body)
				return fmt.Errorf("trigger failed: %s: %s", resp2.Status, string(b2))
			}
		}
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("trigger failed: %s: %s", resp.Status, string(b))
}

func fetchJobBuilds(cfg *JenkinsConfig, job string, limit int) ([]buildInfo, error) {
	// Cookie jar is not strictly required for GET, but harmless and consistent
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 30 * time.Second, Jar: jar}
	// request limited tree
	endpoint := fmt.Sprintf("%s/api/json?tree=builds[number,result,timestamp,duration,building]{,%d}", buildJobPathBase(cfg, job), limit)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", basicAuthHeader(cfg.Username, cfg.Secret))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("job info failed: %s: %s", resp.Status, string(b))
	}
	var ji jobInfo
	if err := json.NewDecoder(resp.Body).Decode(&ji); err != nil {
		return nil, err
	}
	return ji.Builds, nil
}

func fetchBuildLog(cfg *JenkinsConfig, job string, buildNum int64) (string, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 60 * time.Second, Jar: jar}
	endpoint := fmt.Sprintf("%s/%d/consoleText", buildJobPathBase(cfg, job), buildNum)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", basicAuthHeader(cfg.Username, cfg.Secret))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("log fetch failed: %s: %s", resp.Status, string(b))
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ensureJenkinsConfig() (*JenkinsConfig, error) {
	cfg, err := loadJenkinsConfig()
	if err == nil {
		return cfg, nil
	}
	return askForJenkinsConfig()
}

// clearScreen clears the terminal to keep each screen clean.
// It uses ANSI escape sequences which are supported by modern terminals on
// Windows 10+ and most POSIX systems. As a simple fallback, it prints
// several new lines if ANSI is not handled by the terminal.
func jenkinsClearScreen() {
	// ANSI clear: clear entire screen and move cursor to home
	fmt.Print("\033[2J\033[H")
}

func jenkinsMainMenu() (string, error) {
	green := color.New(color.FgGreen).SprintFunc()
	mag := color.New(color.FgMagenta).SprintFunc()
	title := fmt.Sprintf("%s %s", mag("Jenkins"), green("CLI"))
	prompt := promptui.Select{
		Label: title + " — select action",
		Items: []string{"Trigger build", "View history", "View build log", "Configure", "Exit"},
		Size:  5,
	}
	_, v, err := prompt.Run()
	return v, err
}

func readKeyValueParams() map[string]string {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("Enter parameters as key=value, one per line. Leave blank to finish."))
	res := map[string]string{}
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("param> ")
		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			break
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			color.Red("Skip: expected key=value")
			continue
		}
		res[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return res
}

func jenkinsConfigureMenu() (*JenkinsConfig, error) {
	return askForJenkinsConfig()
}

func jenkinsCliMain() {
	jenkinsClearScreen()
	color.Cyan("✨ Jenkins CLI — Trigger builds, view history and logs. Hello!")
	cfg, err := ensureJenkinsConfig()
	if err != nil {
		color.Red("Unable to load configuration: %v", err)
		return
	}
	for {
		// New screen for main menu
		jenkinsClearScreen()
		choice, err := jenkinsMainMenu()
		if err != nil {
			fmt.Println()
			return
		}
		switch choice {
		case "Trigger build":
			jenkinsClearScreen()
			job, ok := selectJob(cfg)
			if !ok {
				continue
			}
			params := map[string]string{}
			ynSel := promptui.Select{Label: "Pass parameters?", Items: []string{"No", "Yes"}}
			_, ans, _ := ynSel.Run()
			if ans == "Yes" {
				params = readKeyValueParams()
			}
			if err := triggerBuild(cfg, job, params); err != nil {
				color.Red("Trigger failed: %v", err)
			} else {
				color.Green("✔ Build request sent for job %s", job)
			}

		case "View history":
			jenkinsClearScreen()
			job, ok := selectJob(cfg)
			if !ok {
				continue
			}
			limitPrompt := promptui.Prompt{Label: "Number of records (default 10)", Default: "10"}
			ls, _ := limitPrompt.Run()
			limit, _ := strconv.Atoi(ls)
			if limit <= 0 {
				limit = 10
			}
			builds, err := fetchJobBuilds(cfg, job, limit)
			if err != nil {
				color.Red("Failed to get history: %v", err)
				continue
			}
			if len(builds) == 0 {
				color.Yellow("No builds yet")
				continue
			}
			bold := color.New(color.Bold).SprintFunc()
			for _, b := range builds {
				t := time.UnixMilli(b.Timestamp).Local()
				statusColor := color.New(color.FgWhite)
				switch strings.ToUpper(b.Result) {
				case "SUCCESS":
					statusColor = color.New(color.FgGreen)
				case "FAILURE", "ABORTED":
					statusColor = color.New(color.FgRed)
				default:
					if b.Building {
						statusColor = color.New(color.FgYellow)
					}
				}
				fmt.Printf("#%s  %s  %s  dur=%ds\n", bold(b.Number), statusColor.Sprintf("%v", valueOr(b.Result, func() string {
					if b.Building {
						return "BUILDING"
					}
					return "?"
				}())), t.Format("2006-01-02 15:04:05"), b.Duration/1000)
			}

		case "View build log":
			jenkinsClearScreen()
			job, ok := selectJob(cfg)
			if !ok {
				continue
			}
			numPrompt := promptui.Prompt{Label: "Build number"}
			ns, err := numPrompt.Run()
			if err != nil {
				continue
			}
			n, err := strconv.ParseInt(strings.TrimSpace(ns), 10, 64)
			if err != nil {
				color.Red("Invalid build number")
				continue
			}
			log, err := fetchBuildLog(cfg, job, n)
			if err != nil {
				color.Red("Failed to fetch log: %v", err)
				continue
			}
			purple := color.New(color.FgHiMagenta).SprintFunc()
			fmt.Println(purple("======== BUILD LOG START ========"))
			fmt.Println(log)
			fmt.Println(purple("========  BUILD LOG END  ========"))

		case "Configure":
			jenkinsClearScreen()
			newCfg, err := jenkinsConfigureMenu()
			if err != nil {
				color.Red("Configuration failed: %v", err)
			} else {
				cfg = newCfg
			}

		case "Exit":
			color.Cyan("Goodbye 👋")
			return
		}
	}
}

func valueOr[T any](v T, alt T) T {
	// If v is string and empty -> alt
	switch any(v).(type) {
	case string:
		if any(v).(string) == "" {
			return alt
		}
	}
	return v
}

// fetchJobs retrieves job list from Jenkins root using the JSON API.
// It returns a slice of job full names (or names if fullName is empty).
type jobsResponse struct {
	Jobs []struct {
		Name     string `json:"name"`
		FullName string `json:"fullName"`
	} `json:"jobs"`
}

func fetchJobs(cfg *JenkinsConfig) ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := cfg.JenkinsURL + "/api/json?tree=jobs[name,fullName]"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", basicAuthHeader(cfg.Username, cfg.Secret))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jobs fetch failed: %s: %s", resp.Status, string(b))
	}
	var jr jobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		return nil, err
	}
	res := make([]string, 0, len(jr.Jobs))
	for _, j := range jr.Jobs {
		name := j.FullName
		if name == "" {
			name = j.Name
		}
		if name != "" {
			res = append(res, name)
		}
	}
	return res, nil
}

// buildJobPathBase builds the base URL path for a (possibly nested) job full name.
// Jenkins expects /job/seg1/job/seg2 for nested jobs.
func buildJobPathBase(cfg *JenkinsConfig, fullName string) string {
	if strings.TrimSpace(fullName) == "" {
		return cfg.JenkinsURL + "/job/" // fallback
	}
	parts := strings.Split(fullName, "/")
	esc := make([]string, 0, len(parts)*2)
	for i, p := range parts {
		_ = i
		esc = append(esc, "job")
		esc = append(esc, url.PathEscape(p))
	}
	return cfg.JenkinsURL + "/" + strings.Join(esc, "/")
}

// selectJob shows a list of jobs fetched from Jenkins and returns the chosen job full name.
func selectJob(cfg *JenkinsConfig) (string, bool) {
	jobs, err := fetchJobs(cfg)
	if err != nil {
		color.Red("Failed to fetch jobs: %v", err)
		return "", false
	}
	if len(jobs) == 0 {
		color.Yellow("No jobs found")
		return "", false
	}
	sel := promptui.Select{Label: "Select job", Items: jobs, Size: 10}
	_, v, err := sel.Run()
	if err != nil {
		return "", false
	}
	return v, true
}
