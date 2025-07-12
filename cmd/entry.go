package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"subcut/parser"
)

type (
	CommandFunc func(args []string)

	FlagInfo struct {
		Name        string
		Description string
	}

	CommandInfo struct {
		Description string
		Function    CommandFunc
		Flags       []FlagInfo
	}
)

var commands map[string]CommandInfo

func init() {
	commands = map[string]CommandInfo{
		"run": {
			Description: "Takes the filepath of program, and executes it",
			Function:    Run,
			Flags: []FlagInfo{
				{
					Name:        "-f",
					Description: "program file path",
				},
			},
		},
		"help": {
			Description: "Prints the usage of all commands",
			Function:    Help,
			Flags:       []FlagInfo{},
		},
	}
}

func Help(args []string) {
	if len(args) < 1 {
		// show the whole help catalog
		printResult := "\n\033[1;35mSupported Commands:\033[0m\n\n"

		for name, cmd := range commands {
			printResult += fmt.Sprintf("  \033[1;36m%v\033[0m\n", name)
			printResult += fmt.Sprintf("    \033[1;37mDescription:\033[0m \033[0;37m%v\033[0m\n", cmd.Description)

			if len(cmd.Flags) > 0 {
				printResult += "    \033[1;37mFlags:\033[0m\n"
				for _, flag := range cmd.Flags {
					printResult += fmt.Sprintf("      \033[1;33m%v\033[0m - \033[0;37m%v\033[0m\n", flag.Name, flag.Description)
				}
			}
			printResult += "\n"
		}

		fmt.Println(printResult)
	} else if len(args) == 1 {
		// print the help of the specified commands
		cmdName := args[0]

		// check if command is supported or not
		if _, ok := commands[cmdName]; !ok {
			fmt.Println("ERROR: provided command, isn't supported")
			return
		}

		cmd := commands[cmdName]

		printResult := fmt.Sprintf("\n\033[1;35mCommand:\033[0m \033[1;36m%v\033[0m\n", cmdName)
		printResult += fmt.Sprintf("\033[1;37mDescription:\033[0m \033[0;37m%v\033[0m\n", cmd.Description)

		if len(cmd.Flags) > 0 {
			printResult += fmt.Sprintln("\033[1;37mFlags:\033[0m")
			for _, flag := range cmd.Flags {
				printResult += fmt.Sprintf("  \033[1;33m%v\033[0m - \033[0;37m%v\033[0m\n", flag.Name, flag.Description)
			}
		} else {
			printResult += "\033[0;37m(No flags available)\033[0m\n"
		}

		fmt.Println(printResult)
	}
}

func Run(args []string) {
	fileTarget := ""
	if len(args) < 2 {
		if args[0] != "-f" {
			fmt.Println("ERROR: provide the filepath flag -f to assign the path to it")
			return
		}

		if len(args[1]) <= 0 {
			fmt.Println("ERROR: provide the filepath to the subcut script")
			return
		}
	}

	fileTarget = args[1]

	// open the file target in this case
	if len(fileTarget) <= 0 {
		fmt.Println("ERROR: provide a valid filepath")
		return
	}

	osPath, _ := os.Getwd()
	targetFile := filepath.Join(osPath, fileTarget)

	byteContent, err := os.ReadFile(targetFile)

	if err != nil {
		fmt.Println(err)
		return
	}

	content := string(byteContent)

	lexer := parser.NewLexer(targetFile, content)
	tokens := lexer.Tokenize()

	// tokens
	fmt.Println(tokens)
	filename, _ := os.Stat(targetFile)
	p := parser.NewParser(tokens, filename.Name())
	ast := p.Parse()

	if ast == nil {
		return
	}

	fmt.Println("Parsed successfully")
	fmt.Println(ast)
}

func Execute() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: at least provide command name to kick off the cli")
		return
	}

	name := os.Args[1]
	args := os.Args[2:]

	if _, ok := commands[name]; !ok {
		fmt.Printf("ERROR: unknown command %v, check help for manual.\n", name)
		return
	}

	commands[name].Function(args)
}
