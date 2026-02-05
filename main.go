package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mattn/go-colorable"
	"github.com/neilotoole/jsoncolor"
	"gopkg.in/yaml.v3"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// Config represents the Hepi configuration file structure.
type Config struct {
	Environments yaml.Node           `yaml:"environments"`
	Requests     yaml.Node           `yaml:"requests"`
	Groups       map[string][]string `yaml:"groups"`
}

// Request represents an individual API request definition.
type Request struct {
	Method      string                 `yaml:"method"`
	URL         string                 `yaml:"url"`
	Description string                 `yaml:"description"`
	Headers     map[string]string      `yaml:"headers"`
	Params      map[string]interface{} `yaml:"params"`
	JSON        map[string]interface{} `yaml:"json"`
	Form        map[string]interface{} `yaml:"form"`
	Files       map[string]string      `yaml:"files"`
}

// Runner manages the execution of API requests.
type Runner struct {
	Config      Config
	EnvName     string
	Environment map[string]interface{}
	State       map[string]interface{}
	HTTPClient  *http.Client
	ShowHeaders bool
	StateFile   string
}

func main() {
	godotenv.Load()

	var envName string
	flag.StringVar(&envName, "env", "", "Environment to use")

	var filePath string
	flag.StringVar(&filePath, "file", "", "Path to the YAML file")

	var statePath string
	flag.StringVar(&statePath, "state", ".hepi.json", "Path to state file")
	reqNames := flag.String("req", "", "Comma-separated list of request names to execute")
	groupName := flag.String("group", "", "Group to execute")
	showHeaders := flag.Bool("headers", false, "Display response headers")
	timeout := flag.Duration("timeout", 10*time.Second, "Request timeout duration")
	flag.Parse()

	if filePath == "" {
		fmt.Printf("Error: -file is required\n\n")
		fmt.Printf("Usage: %s -env <environment> -file <file_path> [options]\n", os.Args[0])
		os.Exit(1)
	}

	runner, err := NewRunner(filePath, envName, statePath, *timeout)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	runner.ShowHeaders = *showHeaders

	if *groupName == "" && *reqNames == "" {
		fmt.Printf("Error: -group or -req is required\n\n")
		runner.PrintHelp()
		os.Exit(0)
	}

	if envName == "" {
		runner.PrintHelp()
		return
	}

	if *groupName != "" {
		if err := runner.ExecuteGroup(*groupName); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	if *reqNames != "" {
		if err := runner.ExecuteRequests(*reqNames); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}

// NewRunner initializes a new Hepi runner.
func NewRunner(filePath, envName, stateFile string, timeout time.Duration) (*Runner, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%sfailed to read file: %w%s", colorRed, err, colorReset)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%sfailed to parse YAML: %w%s", colorRed, err, colorReset)
	}

	if config.Environments.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%senvironments must be a mapping%s", colorRed, colorReset)
	}

	selectedEnvName := envName
	var selectedEnv map[string]interface{}

	if envName != "" {
		found := false
		var availableEnvs []string
		for i := 0; i < len(config.Environments.Content); i += 2 {
			name := config.Environments.Content[i].Value
			availableEnvs = append(availableEnvs, name)
			if name == envName {
				if err := config.Environments.Content[i+1].Decode(&selectedEnv); err != nil {
					return nil, fmt.Errorf("%sfailed to decode environment %q: %w%s", colorRed, envName, err, colorReset)
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%senvironment %q not found\nAvailable environments:\n- %s%s", colorRed, envName, strings.Join(availableEnvs, "\n- "), colorReset)
		}
	}

	return &Runner{
		Config:      config,
		EnvName:     selectedEnvName,
		Environment: selectedEnv,
		State:       loadState(selectedEnvName, stateFile),
		StateFile:   stateFile,
		HTTPClient:  &http.Client{Timeout: timeout},
	}, nil
}

// ExecuteGroup runs all requests in the specified group.
func (r *Runner) ExecuteGroup(groupName string) error {
	group, ok := r.Config.Groups[groupName]
	if !ok {
		return fmt.Errorf("%sgroup %q not found%s", colorRed, groupName, colorReset)
	}

	for _, reqName := range group {
		if err := r.ExecuteRequests(reqName); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteRequests runs the specified requests.
func (r *Runner) ExecuteRequests(reqNames string) error {
	filter := make(map[string]bool)
	for _, name := range strings.Split(reqNames, ",") {
		filter[strings.TrimSpace(name)] = true
	}

	// Validate that all requested requests exist
	foundRequests := make(map[string]bool)

	requestsNode := r.Config.Requests
	if requestsNode.Kind != yaml.MappingNode {
		return fmt.Errorf("%srequests must be a mapping%s", colorRed, colorReset)
	}

	for i := 0; i < len(requestsNode.Content); i += 2 {
		nameNode := requestsNode.Content[i]
		valNode := requestsNode.Content[i+1]

		name := nameNode.Value
		if !filter[name] {
			continue
		}
		foundRequests[name] = true

		var req Request
		if err := valNode.Decode(&req); err != nil {
			if strings.Contains(err.Error(), "invalid map key") {
				return fmt.Errorf("%sfailed to decode request %q: %w\n%sHint: Check for unquoted template variables like {{foo}} used as values%s", colorRed, name, err, colorYellow, colorReset)
			}
			return fmt.Errorf("%sfailed to decode request %q: %w%s", colorRed, name, err, colorReset)
		}

		fmt.Printf("\n%s--- %s[%s]%s %s ---%s\n", colorBold, colorCyan, name, colorReset, req.Description, colorReset)
		if err := r.executeRequest(name, req); err != nil {
			return err
		}
	}

	var missing []string
	for req := range filter {
		if !foundRequests[req] {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%srequests not found: %s%s", colorRed, strings.Join(missing, ", "), colorReset)
	}

	return nil
}

func (r *Runner) executeRequest(name string, req Request) error {
	rawURL := r.substitute(req.URL)

	// Handle query parameters
	if req.Params != nil {
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("%sfailed to parse URL %q: %w%s", colorRed, rawURL, err, colorReset)
		}
		q := u.Query()
		params := r.substituteMap(req.Params)
		for k, v := range params {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
		rawURL = u.String()
	}

	methodColor := colorCyan
	switch req.Method {
	case "GET":
		methodColor = colorGreen
	case "POST":
		methodColor = colorCyan
	case "PUT", "PATCH":
		methodColor = colorYellow
	case "DELETE":
		methodColor = colorRed
	}

	fmt.Printf("%s%s%s %s\n", methodColor, req.Method, colorReset, rawURL)

	var bodyReader io.Reader
	var contentType string

	if req.JSON != nil {
		jsonBody := r.substituteMap(req.JSON)
		data, _ := json.Marshal(jsonBody)
		bodyReader = bytes.NewReader(data)
		contentType = "application/json"
	} else if req.Files != nil {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add form fields
		if req.Form != nil {
			form := r.substituteMap(req.Form)
			for k, v := range form {
				_ = writer.WriteField(k, fmt.Sprintf("%v", v))
			}
		}

		// Add files
		for field, path := range req.Files {
			substitutedPath := r.substitute(path)
			file, err := os.Open(substitutedPath)
			if err != nil {
				return fmt.Errorf("%sfailed to open file %q: %w%s", colorRed, substitutedPath, err, colorReset)
			}
			defer file.Close()

			part, err := writer.CreateFormFile(field, substitutedPath)
			if err != nil {
				return fmt.Errorf("%sfailed to create form file for %q: %w%s", colorRed, field, err, colorReset)
			}
			_, _ = io.Copy(part, file)
		}

		writer.Close()
		bodyReader = body
		contentType = writer.FormDataContentType()
	} else if req.Form != nil {
		formData := url.Values{}
		form := r.substituteMap(req.Form)
		for k, v := range form {
			formData.Set(k, fmt.Sprintf("%v", v))
		}
		bodyReader = strings.NewReader(formData.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	httpReq, err := http.NewRequest(req.Method, rawURL, bodyReader)
	if err != nil {
		return fmt.Errorf("%sfailed to create HTTP request: %w%s", colorRed, err, colorReset)
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, r.substitute(v))
	}

	startTime := time.Now()
	resp, err := r.HTTPClient.Do(httpReq)
	if err != nil {
		if os.IsTimeout(err) {
			return fmt.Errorf("%srequest timed out after %v%s", colorRed, r.HTTPClient.Timeout, colorReset)
		}
		return fmt.Errorf("%srequest failed: %w%s", colorRed, err, colorReset)
	}
	duration := time.Since(startTime)
	defer resp.Body.Close()

	statusColor := colorRed
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		statusColor = colorGreen
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		statusColor = colorYellow
	}

	fmt.Printf("Status: %s%s%s (took %s%v%s)\n", statusColor, resp.Status, colorReset, colorYellow, duration.Round(time.Millisecond), colorReset)

	if r.ShowHeaders {
		fmt.Printf("\n%sHeaders:%s\n", colorBold, colorReset)
		for k, v := range resp.Header {
			fmt.Printf("  %s%s%s: %s\n", colorCyan, k, colorReset, strings.Join(v, ", "))
		}
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%sfailed to read response body: %w%s", colorRed, err, colorReset)
	}

	if len(respData) > 0 {
		var result interface{}
		if err := json.Unmarshal(respData, &result); err == nil {
			result = decodeRecursive(result)
			r.State[name] = result
			r.saveState()
			fmt.Printf("\n%sResponse:%s\n", colorBold, colorReset)

			var enc *jsoncolor.Encoder
			if jsoncolor.IsColorTerminal(os.Stdout) {
				out := colorable.NewColorable(os.Stdout)
				enc = jsoncolor.NewEncoder(out)
				enc.SetColors(jsoncolor.DefaultColors())
			} else {
				enc = jsoncolor.NewEncoder(os.Stdout)
			}

			enc.SetIndent("", "  ")
			if err := enc.Encode(result); err != nil {
				fmt.Println(result)
			}
		} else {
			fmt.Printf("\n%sResponse (non-JSON):%s\n", colorBold, colorReset)
			fmt.Println(string(respData))
		}
	}

	return nil
}

func (r *Runner) substitute(s string) string {
	// 1. Handle [[dynamic]] placeholders using the Generators map and oneof support
	genRegex := regexp.MustCompile(`\[\[(.*?)\]\]`)
	s = genRegex.ReplaceAllStringFunc(s, func(match string) string {
		tag := strings.Trim(match[2:len(match)-2], " ")

		// Handle [[oneof: a, b, c]]
		if strings.HasPrefix(tag, "oneof:") {
			parts := strings.Split(tag[6:], ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[rand.Intn(len(parts))])
			}
		}

		// Handle Generators map
		if gen, ok := Generators[tag]; ok {
			return gen()
		}

		// Fallback for random_ prefix if not already present
		if gen, ok := Generators["random_"+tag]; ok {
			return gen()
		}

		return match
	})

	// 2. Handle {{variables}}
	re := regexp.MustCompile(`{{(.*?)}}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		key := strings.Trim(match[2:len(match)-2], " ")

		// Priority 1: System Environment Variables
		if val, exists := os.LookupEnv(key); exists {
			return val
		}

		// Priority 2: Config Environment Variables
		if val, ok := r.Environment[key]; ok {
			return fmt.Sprintf("%v", val)
		}

		// Priority 3: Previous Request Results
		parts := strings.Split(key, ".")
		if len(parts) > 1 {
			if res, ok := r.State[parts[0]]; ok {
				return getValueFromMap(res, parts[1:])
			}
		}

		return match
	})
}

func (r *Runner) substituteMap(m map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case string:
			res[k] = r.substitute(val)
		case map[string]interface{}:
			res[k] = r.substituteMap(val)
		case []interface{}:
			res[k] = r.substituteSlice(val)
		default:
			res[k] = v
		}
	}
	return res
}

func (r *Runner) substituteSlice(s []interface{}) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case string:
			res[i] = r.substitute(val)
		case map[string]interface{}:
			res[i] = r.substituteMap(val)
		case []interface{}:
			res[i] = r.substituteSlice(val)
		default:
			res[i] = v
		}
	}
	return res
}

func (r *Runner) PrintHelp() {
	fmt.Printf("Hepi - REST API Tester\n")
	fmt.Printf("https://github.com/mitjafelicijan/hepi\n\n")
	fmt.Println("Available Environments:")
	if r.Config.Environments.Kind == yaml.MappingNode {
		for i := 0; i < len(r.Config.Environments.Content); i += 2 {
			name := r.Config.Environments.Content[i].Value
			fmt.Printf("  - %s\n", name)
		}
	}

	fmt.Println("\nAvailable Requests:")
	requestsNode := r.Config.Requests
	if requestsNode.Kind == yaml.MappingNode {
		maxLen := 0
		for i := 0; i < len(requestsNode.Content); i += 2 {
			name := requestsNode.Content[i].Value
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}

		for i := 0; i < len(requestsNode.Content); i += 2 {
			name := requestsNode.Content[i].Value
			valNode := requestsNode.Content[i+1]
			var req Request
			_ = valNode.Decode(&req)
			fmt.Printf("  - %-*s  %s\n", maxLen+2, name, req.Description)
		}
	}

	fmt.Println("\nAvailable Groups:")
	for name, reqs := range r.Config.Groups {
		fmt.Printf("  - %s (%s)\n", name, strings.Join(reqs, ", "))
	}

	fmt.Printf("\nUsage:\n  %s -env <environment> -file <file_path> -req <request1,request2,...> -group <group_name> -headers\n", os.Args[0])
}

func loadState(envName, stateFile string) map[string]interface{} {
	allStates := make(map[string]map[string]interface{})
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return make(map[string]interface{})
	}
	json.Unmarshal(data, &allStates)

	if res, ok := allStates[envName]; ok {
		return res
	}
	return make(map[string]interface{})
}

func (r *Runner) saveState() {
	allStates := make(map[string]map[string]interface{})
	data, err := os.ReadFile(r.StateFile)
	if err == nil {
		json.Unmarshal(data, &allStates)
	}

	allStates[r.EnvName] = r.State

	output, err := json.MarshalIndent(allStates, "", "  ")
	if err != nil {
		log.Printf("failed to marshal state: %v", err)
		return
	}
	err = os.WriteFile(r.StateFile, output, 0644)
	if err != nil {
		log.Printf("failed to save state: %v", err)
	}
}

func getValueFromMap(data interface{}, path []string) string {
	for _, part := range path {
		if m, ok := data.(map[string]interface{}); ok {
			data = m[part]
		} else if s, ok := data.([]interface{}); ok {
			idx, err := strconv.Atoi(part)
			if err == nil && idx >= 0 && idx < len(s) {
				data = s[idx]
			} else {
				return fmt.Sprintf("{{MISSING:%s}}", part)
			}
		} else {
			return fmt.Sprintf("{{MISSING:%s}}", part)
		}
	}
	return fmt.Sprintf("%v", data)
}

func decodeRecursive(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			v[k] = decodeRecursive(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = decodeRecursive(val)
		}
		return v
	case string:
		// Try to decode if it looks like JSON (object or array)
		trimmed := strings.TrimSpace(v)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var decoded interface{}
			if err := json.Unmarshal([]byte(v), &decoded); err == nil {
				return decodeRecursive(decoded)
			}
		}
		return v
	default:
		return v
	}
}
