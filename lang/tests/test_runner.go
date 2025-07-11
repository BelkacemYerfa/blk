package tests

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"subcut/parser"
)

type TestResult struct {
	File   string
	Source string
	Pass   bool
	Error  string
	Tokens string
	AST    string
}

func TestRunner() {
	osPath, _ := os.Getwd()
	path := filepath.Join(osPath, "./tests")

	files, err := os.ReadDir(path)

	if err != nil {
		fmt.Println(err)
		return
	}

	var results []TestResult

	for _, file := range files {
		if file.IsDir() {
			fmt.Println("ERROR: dirs are not allowed, only files")
			return
		}

		filePath := filepath.Join(path, file.Name())
		fileExt := filepath.Ext(filePath)

		if fileExt != ".subcut" {
			continue
		}

		// Read the corresponding golden file
		baseName := file.Name()[:len(file.Name())-len(fileExt)]
		goldenFilePath := filepath.Join(path, baseName+".json")

		expectedResults := make(map[string]any)

		if _, err := os.Stat(goldenFilePath); err == nil {
			jsonGoldenFile, err := os.ReadFile(goldenFilePath)
			if err != nil {
				fmt.Println("Error reading golden file:", err)
				continue
			}

			err = json.Unmarshal(jsonGoldenFile, &expectedResults)
			if err != nil {
				fmt.Println("Error parsing golden file:", err)
				continue
			}
		}

		subcutContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}

		content := string(subcutContent)

		result := TestResult{
			File:   file.Name(),
			Source: content,
			Pass:   true,
		}

		lexer := parser.NewLexer(filePath, content)
		tokens := lexer.Tokenize()

		// marshal the tokens
		tokensJson, err := json.Marshal(tokens)
		if err != nil {
			fmt.Println(err)
			return
		}

		var tokensInterface interface{}
		err = json.Unmarshal(tokensJson, &tokensInterface)
		if err != nil {
			fmt.Println(err)
			return
		}

		if expectedTokens, exists := expectedResults["tokens"]; exists {
			if err := deepJsonCheck(expectedTokens, tokensInterface); err != nil {
				result.Pass = false
				result.Error = fmt.Sprintf("Token Mismatch: %v", err)
			}
		}

		tokenBytes, _ := json.MarshalIndent(tokensInterface, "", "  ")
		result.Tokens = string(tokenBytes)

		parser := parser.NewParser(tokens, "")
		ast := parser.Parse()

		if ast != nil {
			astBytes, _ := json.MarshalIndent(ast, "", "  ")
			result.AST = string(astBytes)

			var astInterface interface{}
			err = json.Unmarshal(astBytes, &astInterface)
			if err != nil {
				fmt.Println(err)
				return
			}

			if expectedAST, exists := expectedResults["ast"]; exists {
				if err := deepJsonCheck(expectedAST, astInterface); err != nil {
					result.Pass = false
					result.Error += "\nAST Mismatch: " + err.Error()
				}
			}
		} else {
			result.Pass = false
			result.Error += "\nParser returned nil AST"
		}

		results = append(results, result)
	}
	writeHTMLResult(results, "test_report.html")
}

func deepJsonCheck(expected, actual interface{}) error {
	if !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("expected %v, got %v", expected, actual)
	}
	return nil
}

func writeHTMLResult(results []TestResult, filename string) {
	htmlResult := `<!DOCTYPE htmlResult>
<htmlResult>
<head>
	<title>Test Results</title>
	<style>
		body { font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; margin: 20px; background: #1e1e1e; color: #d4d4d4; }
		.test-result { border: 1px solid #404040; margin: 15px 0; padding: 20px; border-radius: 8px; background: #252526; }
		.pass { border-left: 4px solid #4CAF50; }
		.fail { border-left: 4px solid #f44336; }
		.status { font-weight: bold; font-size: 14px; }
		.pass .status { color: #4CAF50; }
		.fail .status { color: #f44336; }
		.error {
			color: #f44336;
			margin: 15px 0;
			background: #2d1b1b;
			padding: 15px;
			border-radius: 5px;
			border-left: 4px solid #f44336;
			font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
			white-space: pre-wrap;
			line-height: 1.4;
		}
		.error-title { color: #ff6b6b; font-weight: bold; margin-bottom: 10px; }
		.code { background: #2d2d30; padding: 15px; border-radius: 5px; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; white-space: pre-wrap; overflow-x: auto; border: 1px solid #404040; }
		.section { margin: 15px 0; }
		.section-title { font-weight: bold; margin-bottom: 8px; color: #569cd6; }
		summary { cursor: pointer; font-weight: bold; color: #569cd6; padding: 8px 0; }
		details { margin: 8px 0; }
		details[open] summary { margin-bottom: 10px; }
		h1 { color: #569cd6; border-bottom: 2px solid #404040; padding-bottom: 10px; }
		h3 { color: #d4d4d4; margin-top: 0; }
		.file-name { color: #9cdcfe; }
	</style>
</head>
<body>
	<h1>Test Results</h1>
`

	for _, result := range results {
		status := "PASS"
		class := "pass"
		if !result.Pass {
			status = "FAIL"
			class = "fail"
		}

		htmlResult += fmt.Sprintf(`
	<div class="test-result %s">
		<h3><span class="file-name">%s</span> - <span class="status">%s</span></h3>
		<div class="section">
			<div class="section-title">Source Code:</div>
			<div class="code">%s</div>
		</div>
`, class, result.File, status, result.Source)
		if result.Error != "" {
			htmlResult += fmt.Sprintf(`
		<div class="error">
			<div class="error-title">❌ Test Failure Details:</div>
			<div style="margin-top: 10px; line-height: 1.6;">%s</div>
		</div>
`, html.EscapeString(result.Error))
		}

		if result.Tokens != "" {
			htmlResult += fmt.Sprintf(`
		<details>
			<summary>🔍 View Tokens</summary>
			<div class="code">%s</div>
		</details>
`, result.Tokens)
		}

		if result.AST != "" {
			htmlResult += fmt.Sprintf(`
		<details>
			<summary>🌳 View AST</summary>
			<div class="code">%s</div>
		</details>
`, result.AST)
		}

		htmlResult += "    </div>\n"
	}

	htmlResult += `
</body>
</htmlResult>`

	err := os.WriteFile(filename, []byte(htmlResult), 0644)
	if err != nil {
		fmt.Printf("Error writing HTMLResult file: %v\n", err)
		return
	}

	fmt.Printf("Test results written to %s\n", filename)
}
